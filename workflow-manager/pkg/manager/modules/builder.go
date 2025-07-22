package modules

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coredgeio/compass/pkg/apply"
	"github.com/coredgeio/compass/pkg/errors"
	inframanager "github.com/coredgeio/compass/pkg/infra/manager"
	"github.com/coredgeio/compass/pkg/infra/notifier"
	"github.com/coredgeio/orbiter-registry/api/registry"

	"github.com/coredgeio/workflow-manager/pkg/module/builder"
	"github.com/coredgeio/workflow-manager/pkg/runtime"
	"github.com/coredgeio/workflow-manager/pkg/runtime/baseimage"
	"github.com/coredgeio/workflow-manager/pkg/runtime/module"
)

const (
	// Notifier client name for module build manager
	ModuleBuildManagerName = "WorkflowModuleBuilder"
)

var (
	Namespace string
	clientset *kubernetes.Clientset
	k8sClient k8sclient.Client
	podClient clientCoreV1.PodInterface
)

type ModuleReconciler struct {
	notifier.Client
	mgr *ModuleBuilder
}

func (r *ModuleReconciler) Reconcile(rkey interface{}) (*notifier.Result, error) {
	key, ok := rkey.(module.ModuleKey)
	if !ok {
		log.Fatalln("Received key not of type ModuleKey in", ModuleBuildManagerName, rkey)
	}
	if !r.mgr.IsOwnershipAcquired() {
		return &notifier.Result{}, nil
	}

	entry, err := r.mgr.table.Find(&key)
	if err != nil {
		if !errors.IsNotFound(err) {
			// something went wrong, should not happen, we can safely ignore for now
			log.Println("failed to find the Module entry", key)
			return nil, err
		}
		return &notifier.Result{}, nil
	}

	if entry.IsDeleted {
		bName := "m-" + entry.Id.String()
		// check if the pod is completed and corresponding status
		_, err := podClient.Get(context.TODO(), bName, metav1.GetOptions{})
		if err == nil {
			_ = podClient.Delete(context.TODO(), bName, metav1.DeleteOptions{})
			// wait for the builder pod to complete
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}

		req := &registry.DeleteRepoReq{
			Domain: "default",
			Name:   "catalog",
			Repo:   bName,
		}
		_, err = r.mgr.regClient.DeleteRepo(req)
		if err != nil {
			log.Println("failed to delete catalog repo", bName, err)
		}
		err = r.mgr.table.Remove(&entry.Key)
		if err != nil {
			log.Println("failed to delete module from runtime", entry.Key)
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}
		return &notifier.Result{}, nil
	}

	if entry.BuildConfig == nil {
		log.Println("skipping module with build config nil", entry.Key)
		return &notifier.Result{}, nil
	}

	if entry.BuildStatus != nil {
		if reflect.DeepEqual(*entry.BuildConfig, *entry.BuildStatus.Config) {
			// since there is no change in config, skip further processing
			// if the build is completed with success or failure, till a
			// config change or rebuild is triggered
			if entry.BuildStatus.Status == module.ModuleBuildCompleted ||
				entry.BuildStatus.Status == module.ModuleBuildFailed {
				return &notifier.Result{}, nil
			}
		}
	}

	status := &module.ModuleBuildStatus{}

	bName := "m-" + entry.Id.String()
	// check if the pod is completed and corresponding status
	pod, err := podClient.Get(context.TODO(), bName, metav1.GetOptions{})
	buildDone := false
	if err != nil {
		log.Println("failed to get builder pod", bName, err)
		baseImage := entry.BuildConfig.BaseImage
		imgUuid, err := uuid.Parse(baseImage)
		// TODO(Prabhjot) we need to decide if we are going to support
		// external base image references here. right now
		// UI will be limiting it
		if err == nil {
			bImage, err := r.mgr.baseImageTbl.FindById(&imgUuid)
			if err != nil {
				log.Println("failed to find base image for module build", baseImage, err)
				return nil, err
			}
			baseImage = bImage.ExternalRef
		}
		dBuilder := builder.DockerFileBuilder{
			BaseImage:   baseImage,
			BuildScript: entry.BuildConfig.BuildScript,
			EntryPoint:  entry.BuildConfig.EntryPoint,
			Files:       []*builder.DockerFileInfo{},
			EnvVars:     entry.BuildConfig.Env,
		}
		if entry.BuildConfig.GitInfo != nil && entry.BuildConfig.GitInfo.Url != "" {
			dBuilder.GitInfo = &builder.DockerGitInfo{
				Url:        entry.BuildConfig.GitInfo.Url,
				GitRef:     entry.BuildConfig.GitInfo.GitRef,
				WorkingDir: entry.BuildConfig.GitInfo.WorkingDir,
			}
		}

		for _, file := range entry.BuildConfig.Files {
			bFile := &builder.DockerFileInfo{
				Name: file.Name,
				Perm: file.Perm,
			}
			dBuilder.Files = append(dBuilder.Files, bFile)
		}
		fileContent, err := dBuilder.GetDockerFile()
		if err != nil {
			log.Println("got error while builder docker file", err)
		} else {
			log.Println("got docker file", fileContent)
		}
		// schedule a build as the pod doesn't exist otherwise
		mBuilder := builder.ModuleBuilder{
			Name:        bName,
			Namespace:   Namespace,
			Files:       []*builder.ModuleFileInfo{},
			DockerFile:  fileContent,
			RegInsecure: true,
			Registry:    r.mgr.regName,
			RegSecret:   "catalog-reg-creds",
		}
		if entry.BuildConfig.GitInfo != nil && entry.BuildConfig.GitInfo.Url != "" {
			mBuilder.GitInfo = &builder.ModuleGitInfo{
				Url:        entry.BuildConfig.GitInfo.Url,
				GitRef:     entry.BuildConfig.GitInfo.GitRef,
				WorkingDir: entry.BuildConfig.GitInfo.WorkingDir,
			}
		}
		for _, file := range entry.BuildConfig.Files {
			mFile := &builder.ModuleFileInfo{
				Name:    file.Name,
				Content: string(file.Content),
			}
			mBuilder.Files = append(mBuilder.Files, mFile)
		}
		objs, err := mBuilder.GetK8sObjects()
		if err != nil {
			log.Println("error generating k8s objects", err)
		} else {
			for _, obj := range objs {
				if err := apply.ApplyObject(context.TODO(), k8sClient, obj); err != nil {
					log.Println("failed to apply object", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
					return nil, err
				}
			}
		}
		status.Status = module.ModuleBuildInProgress
		status.Config = entry.BuildConfig
		update := &module.ModuleEntry{
			Key:         entry.Key,
			BuildStatus: status,
		}
		err = r.mgr.table.Update(update)
		if err != nil {
			log.Println("failed to update build status", entry.Key, err)
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}
	} else {
		logs, err := func() (string, error) {
			podLogOpts := corev1.PodLogOptions{}
			req := podClient.GetLogs(bName, &podLogOpts)
			podLogs, err := req.Stream(context.Background())
			if err != nil {
				log.Println("failed to open pod logs", err)
				return "", err
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				return "", err
			}
			str := buf.String()

			return str, nil
		}()
		if err != nil {
			log.Println("failed to fetch pod logs", err)
		} else {
			status.Logs = logs
		}

		inprogress := false
		switch pod.Status.Phase {
		case "Succeeded":
			buildDone = true
			status.Status = module.ModuleBuildCompleted
			status.BuildTime = time.Now().Unix()
		case "Running", "Pending":
			inprogress = true
			status.Status = module.ModuleBuildInProgress
		default:
			status.Status = module.ModuleBuildFailed
			log.Println("observed a build failure:", bName)
		}

		if entry.BuildStatus == nil || entry.BuildStatus.Config == nil {
			// TODO this may be an issue
			// but required to complete the state correctly
			status.Config = entry.BuildConfig
		} else {
			status.Config = entry.BuildStatus.Config
		}
		update := &module.ModuleEntry{
			Key:         entry.Key,
			BuildStatus: status,
		}
		err = r.mgr.table.Update(update)
		if err != nil {
			log.Println("failed to update build status", entry.Key, err)
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}

		if inprogress {
			// we are still waiting for completion of the running job
			// wait for 10 seconds and retry
			return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
		}

		err = podClient.Delete(context.TODO(), bName, metav1.DeleteOptions{})
		if err != nil {
			log.Println("failed to delete the existing pod", bName)
		}
	}
	log.Println("build done", buildDone)

	return &notifier.Result{}, nil
}

type ModuleBuilder struct {
	inframanager.ManagerImpl
	table        *module.ModuleTable
	baseImageTbl *baseimage.BaseImageVersionTable
	regClient    registry.RegistryApiSdkClient
	regName      string
}

func (m *ModuleBuilder) Start() {
	log.Println(ModuleBuildManagerName, "started")
	r := &ModuleReconciler{
		mgr: m,
	}
	err := m.table.RegisterClient(ModuleBuildManagerName, r)
	if err != nil {
		log.Fatalln(ModuleBuildManagerName, "failed to register ModuleTable", err)
	}
}

func CreateModuleBuilder(regConfig *RegClientConfig, regName string) *ModuleBuilder {
	tbl, err := module.LocateModuleTable()
	if err != nil {
		log.Fatalln("failed locating module table", err)
	}
	baseImageTbl, err := baseimage.LocateBaseImageVersionTable()
	if err != nil {
		log.Fatalln("ModuleBuilder: failed locating base image table", err)
	}
	manager := &ModuleBuilder{
		ManagerImpl: inframanager.ManagerImpl{
			InstanceKey: runtime.ModuleBuildManagerInstanceKey,
		},
		table:        tbl,
		baseImageTbl: baseImageTbl,
		regClient:    regConfig.getRegistryClient(),
		regName:      regName,
	}

	manager.InitImplWithTerminateHandling(manager)

	return manager
}

func init() {
	// get namespace of the pod in which it is running
	bytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalln("failed to get pod namespace ", err)
	}
	Namespace = string(bytes)

	k8sConfig := ctrl.GetConfigOrDie()
	k8sClient, err = k8sclient.New(k8sConfig, k8sclient.Options{})
	if err != nil {
		log.Fatalln("module builder: failed to get k8s client", err)
	}

	// load kube client by config for compass controller
	clientset, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalln("failed to load k8s config for controller ", err)
	}

	podClient = clientset.CoreV1().Pods(Namespace)
}
