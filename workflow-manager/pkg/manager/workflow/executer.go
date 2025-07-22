package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	typedv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coredgeio/compass/pkg/apply"
	"github.com/coredgeio/compass/pkg/errors"
	inframanager "github.com/coredgeio/compass/pkg/infra/manager"
	"github.com/coredgeio/compass/pkg/infra/notifier"

	"github.com/coredgeio/workflow-manager/pkg/runtime"
	"github.com/coredgeio/workflow-manager/pkg/runtime/module"
	"github.com/coredgeio/workflow-manager/pkg/runtime/template"
	wRuntime "github.com/coredgeio/workflow-manager/pkg/runtime/workflow"
	"github.com/coredgeio/workflow-manager/pkg/workflow/builder"
)

const (
	// Notifier client name for module build manager
	WorkflowExecutionManagerName = "WorkflowExecutorManager"
)

var (
	Namespace    string
	k8sClient    k8sclient.Client
	clientset    *versioned.Clientset
	argoWfClient typedv1alpha1.WorkflowInterface
)

type WorkflowReconciler struct {
	notifier.Client
	mgr *WorkflowExecutor
}

func (r *WorkflowReconciler) Reconcile(rkey interface{}) (*notifier.Result, error) {
	key, ok := rkey.(wRuntime.WorkflowKey)
	if !ok {
		log.Fatalln("Received key not of type WorkflowKey in", WorkflowExecutionManagerName, rkey)
	}
	if !r.mgr.IsOwnershipAcquired() {
		return &notifier.Result{}, nil
	}

	entry, err := r.mgr.workflowTbl.Find(&key)
	if err != nil {
		if !errors.IsNotFound(err) {
			// something went wrong, should not happen, we can safely ignore for now
			log.Println("failed to find the Workflow entry", key)
			return nil, err
		}
		return &notifier.Result{}, nil
	}

	if entry.IsDeleted {
		// check if the workflow is completed and corresponding status
		_, err := argoWfClient.Get(context.TODO(), entry.Key.Name, metav1.GetOptions{})
		if err == nil {
			_ = argoWfClient.Delete(context.TODO(), entry.Key.Name, metav1.DeleteOptions{})
			// wait for the workflow to be deleted
			return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
		}
		err = r.mgr.workflowTbl.Remove(&entry.Key)
		if err != nil {
			log.Println("failed to delete module from runtime", entry.Key)
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}
		return &notifier.Result{}, nil
	}

	// check if the workflow is completed and corresponding status
	argoWorkflow, err := argoWfClient.Get(context.TODO(), entry.Key.Name, metav1.GetOptions{})
	if err != nil {
		if entry.State != wRuntime.WorkflowCreated ||
			(entry.Status != nil && entry.Status.State != wRuntime.WorkflowCreated) {
			// something went wrong retry later
			return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
		}
	}

	status := entry.Status
	if status == nil {
		status = &wRuntime.WorkflowStatus{
			State: entry.State,
		}
		// TODO(prabhjot) for backward compatibility
		// can be removed eventually
		if entry.State != wRuntime.WorkflowCreated {
			tempKey := &template.TemplateKey{
				Domain:  entry.Key.Domain,
				Project: entry.Key.Project,
				Name:    entry.Template,
			}
			temp, err := r.mgr.templateTbl.Find(tempKey)
			if err != nil {
				log.Println("failed to find the associated template", entry.Key)
				return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
			}
			for _, n := range temp.Nodes {
				wNode := &wRuntime.WorkflowNode{
					Type:     wRuntime.WorkflowNodeType(n.Type),
					Name:     n.Name,
					NodeId:   n.NodeId,
					ModuleId: n.ModuleId,
					Stage:    n.Stage,
					X:        n.X,
					Y:        n.Y,
					State:    wRuntime.WorkflowCreated,
				}
				status.Nodes = append(status.Nodes, wNode)
			}
			for _, l := range temp.Links {
				wLink := &wRuntime.WorkflowLink{
					Source:    l.Source,
					SourceVar: l.SourceVar,
					Target:    l.Target,
					TargetVar: l.TargetVar,
				}
				status.Links = append(status.Links, wLink)
			}
			update := &wRuntime.WorkflowEntry{
				Key:    entry.Key,
				Status: status,
			}
			err = r.mgr.workflowTbl.Update(update)
			if err != nil {
				log.Println("failed to update workflow status", update.Key, err)
				return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
			}
		}
	}

	switch status.State {
	case wRuntime.WorkflowCreated:
		tempKey := &template.TemplateKey{
			Domain:  entry.Key.Domain,
			Project: entry.Key.Project,
			Name:    entry.Template,
		}
		temp, err := r.mgr.templateTbl.Find(tempKey)
		if err != nil {
			log.Println("failed to find the associated template", entry.Key)
			return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
		}
		wbuilder := &builder.WorkflowBuilder{
			Name:           entry.Key.Name,
			Namespace:      Namespace,
			ServiceAccount: r.mgr.saName,
			Nodes:          map[string]*builder.WorkflowNodesType{},
			Steps:          []*builder.WorkflowStepType{},
		}
		nMap := map[string]*builder.WorkflowStepNode{}
		for _, n := range temp.Nodes {
			wNode := &wRuntime.WorkflowNode{
				Type:     wRuntime.WorkflowNodeType(n.Type),
				Name:     n.Name,
				NodeId:   n.NodeId,
				ModuleId: n.ModuleId,
				Stage:    n.Stage,
				X:        n.X,
				Y:        n.Y,
				State:    wRuntime.WorkflowCreated,
			}
			status.Nodes = append(status.Nodes, wNode)
			if n.Type == template.UserInputNode {
				wNode.State = wRuntime.WorkflowCompleted
				continue
			}
			if len(wbuilder.Steps) <= int(n.Stage) {
				for i := len(wbuilder.Steps); i <= int(n.Stage+1); i++ {
					wbuilder.Steps = append(wbuilder.Steps, &builder.WorkflowStepType{})
				}
			}
			step := wbuilder.Steps[n.Stage]
			node := &builder.WorkflowStepNode{
				NodeId: n.NodeId,
				Module: n.ModuleId,
			}
			step.Nodes = append(step.Nodes, node)
			nMap[n.NodeId] = node
			modNode, ok := wbuilder.Nodes[n.ModuleId]
			if !ok {
				// assuming this will be validated in server
				modUuid, _ := uuid.Parse(n.ModuleId)
				mod, err := r.mgr.moduleTbl.FindById(&modUuid)
				if err != nil {
					log.Println("failed to find the associated module", entry.Key, err)
					return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
				}
				image := r.mgr.regName + "/m-" + n.ModuleId
				name := fmt.Sprintf("m%d", (len(wbuilder.Nodes) + 1))
				modNode = &builder.WorkflowNodesType{
					ModuleName: name,
					Image:      image,
				}
				for k, v := range mod.InputKeys {
					in := &builder.WorkflowInputType{
						Name:  k,
						Value: v.DefaultVal,
					}
					modNode.Inputs = append(modNode.Inputs, in)
				}
				for k, v := range mod.OutputKeys {
					out := &builder.WorkflowOutputType{
						Name:      k,
						ValueFrom: v.ValueFrom,
					}
					modNode.Outputs = append(modNode.Outputs, out)
				}
				wbuilder.Nodes[n.ModuleId] = modNode
			}
			node.Module = modNode.ModuleName
		}
		for _, l := range temp.Links {
			wLink := &wRuntime.WorkflowLink{
				Source:    l.Source,
				SourceVar: l.SourceVar,
				Target:    l.Target,
				TargetVar: l.TargetVar,
			}
			status.Links = append(status.Links, wLink)
			node, ok := nMap[l.Target]
			if !ok {
				continue
			}
			in := &builder.WorkflowStepInputType{
				Name: l.TargetVar,
			}
			node.Inputs = append(node.Inputs, in)
			// if source is not a step that means it is a userInput
			_, ok = nMap[l.Source]
			if !ok {
				value, ok := entry.Inputs[l.Source]
				if !ok {
					log.Println("failed to find user input", l.Source, entry.Key)
					return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
				}
				in.Value = value
			} else {
				in.Source = &builder.WorkflowStepInputSource{
					Source:    l.Source,
					SourceVar: l.SourceVar,
				}
			}
		}

		objs, err := wbuilder.GetK8sObjects()
		if err != nil {
			log.Println("error generating k8s objects for workflow", err)
		} else {
			for _, obj := range objs {
				if err := apply.ApplyObject(context.TODO(), k8sClient, obj); err != nil {
					log.Println("failed to apply object", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
					return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
				}
			}

			status.State = wRuntime.WorkflowScheduled
			update := &wRuntime.WorkflowEntry{
				Key:       entry.Key,
				StartTime: time.Now().Unix(),
				Status:    status,
			}
			err := r.mgr.workflowTbl.Update(update)
			if err != nil {
				log.Println("failed to update workflow status", update.Key, err)
				return &notifier.Result{NotifyAfter: (60 * time.Second)}, nil
			}
		}
	case wRuntime.WorkflowScheduled, wRuntime.WorkflowRunning:
		update := &wRuntime.WorkflowEntry{
			Key:    entry.Key,
			Status: status,
		}

		changed := false
		for _, kn := range argoWorkflow.Status.Nodes {
			for _, n := range status.Nodes {
				if n.NodeId != kn.DisplayName {
					continue
				}

				// if status is already updated skip further update
				if n.State == wRuntime.WorkflowCompleted ||
					n.State == wRuntime.WorkflowFailed {
					continue
				}

				if kn.ID != n.ArgoId {
					n.ArgoId = kn.ID
					changed = true
				}

				if kn.Inputs != nil && len(n.Inputs) == 0 {
					for _, param := range kn.Inputs.Parameters {
						value := ""
						if param.Value != nil {
							value = param.Value.String()
						}
						in := &wRuntime.WorkflowValue{
							Name:  param.Name,
							Value: value,
						}
						n.Inputs = append(n.Inputs, in)
					}
				}
				if kn.Outputs != nil && len(n.Outputs) == 0 {
					for _, param := range kn.Outputs.Parameters {
						value := ""
						if param.Value != nil {
							value = param.Value.String()
						}
						out := &wRuntime.WorkflowValue{
							Name:  param.Name,
							Value: value,
						}
						n.Outputs = append(n.Outputs, out)
					}
				}
				switch kn.Phase {
				case v1alpha1.NodeRunning:
					if n.State != wRuntime.WorkflowRunning {
						n.State = wRuntime.WorkflowRunning
						changed = true
					}
				case v1alpha1.NodeSucceeded:
					if n.State != wRuntime.WorkflowCompleted {
						n.State = wRuntime.WorkflowCompleted
						changed = true
					}
				case v1alpha1.NodeFailed, v1alpha1.NodeError:
					if n.State != wRuntime.WorkflowFailed {
						n.State = wRuntime.WorkflowFailed
						changed = true
					}
				}
			}
		}

		switch argoWorkflow.Status.Phase {
		case v1alpha1.WorkflowRunning:
			if status.State == wRuntime.WorkflowRunning {
				if changed {
					err := r.mgr.workflowTbl.Update(update)
					if err != nil {
						log.Println("failed to update workflow status", update.Key, err)
					}
				}
				return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
			}
			status.State = wRuntime.WorkflowRunning
		case v1alpha1.WorkflowSucceeded:
			status.State = wRuntime.WorkflowCompleted
			update.EndTime = time.Now().Unix()
		case v1alpha1.WorkflowFailed, v1alpha1.WorkflowError:
			status.State = wRuntime.WorkflowFailed
			update.EndTime = time.Now().Unix()
		default:
			// ensure reconciling again after some time
			return &notifier.Result{NotifyAfter: (5 * time.Second)}, nil
		}
		err := r.mgr.workflowTbl.Update(update)
		if err != nil {
			log.Println("failed to update workflow status", update.Key, err)
			return &notifier.Result{NotifyAfter: (10 * time.Second)}, nil
		}
	}

	return &notifier.Result{}, nil
}

type WorkflowExecutor struct {
	inframanager.ManagerImpl
	// service account name to be used for workflows
	saName      string
	regName     string
	moduleTbl   *module.ModuleTable
	templateTbl *template.TemplateTable
	workflowTbl *wRuntime.WorkflowTable
}

func (m *WorkflowExecutor) Start() {
	log.Println(WorkflowExecutionManagerName, "started")
	r := &WorkflowReconciler{
		mgr: m,
	}
	err := m.workflowTbl.RegisterClient(WorkflowExecutionManagerName, r)
	if err != nil {
		log.Fatalln(WorkflowExecutionManagerName, "failed to register Workflow Table", err)
	}
}

// Create Workflow executor with specified service account name to be
// used for created workflows
// along with registry name from where to fetch the images
func CreateWorkflowExecutor(saName, regName string) *WorkflowExecutor {
	moduleTbl, err := module.LocateModuleTable()
	if err != nil {
		log.Fatalln("failed locating module table", err)
	}
	templateTbl, err := template.LocateTemplateTable()
	if err != nil {
		log.Fatalln("failed locating template table", err)
	}
	workflowTbl, err := wRuntime.LocateWorkflowTable()
	if err != nil {
		log.Fatalln("failed locating workflow table", err)
	}
	manager := &WorkflowExecutor{
		ManagerImpl: inframanager.ManagerImpl{
			InstanceKey: runtime.WorkflowExecutorManagerInstanceKey,
		},
		saName:      saName,
		regName:     regName,
		moduleTbl:   moduleTbl,
		templateTbl: templateTbl,
		workflowTbl: workflowTbl,
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

	// load kube client by config for argo CRDs
	clientset, err = versioned.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalln("failed to load k8s config for argo controller ", err)
	}

	argoWfClient = clientset.ArgoprojV1alpha1().Workflows(Namespace)
}
