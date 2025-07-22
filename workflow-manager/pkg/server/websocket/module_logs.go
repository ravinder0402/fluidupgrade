package websocket

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/gorilla/mux"
	gorillaSocket "github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/coredgeio/compass/pkg/infra/websocket"

	"github.com/coredgeio/workflow-manager/pkg/runtime/module"
)

var (
	Namespace string
	clientset *kubernetes.Clientset
	podClient clientCoreV1.PodInterface
)

type ModuleBuildLogsServer struct {
	websocket.Handler
	moduleTable *module.ModuleTable
}

func CreateModuleBuildLogsServer(router *mux.Router) {
	moduleTable, err := module.LocateModuleTable()
	if err != nil {
		log.Fatalln("ModuleBuildLogsServer failed to locate module runtime table", err)
	}

	server := &ModuleBuildLogsServer{
		moduleTable: moduleTable,
	}
	ws := websocket.AllocateServer(server)

	v1Router := router.PathPrefix("/ws/workflow/v1").Subrouter()
	v1Router.HandleFunc("/domain/{domain}/project/{project}/module/{name}/build-logs", ws.ServeHTTP).Methods("GET")
}

func (s *ModuleBuildLogsServer) ServeConnection(ctx *websocket.Context) {
	domain, ok := ctx.Vars["domain"]
	if !ok {
		log.Println("got invalid domain for module build log access")
		writeErrorWebSocket(ctx.Conn, "invalid domain")
		return
	}

	project, ok := ctx.Vars["project"]
	if !ok {
		log.Println("got invalid project for module build log access")
		writeErrorWebSocket(ctx.Conn, "invalid project")
		return
	}

	name, ok := ctx.Vars["name"]
	if !ok {
		log.Println("got invalid module name for build log access")
		writeErrorWebSocket(ctx.Conn, "invalid module name")
		return
	}

	key := &module.ModuleKey{
		Domain:  domain,
		Project: project,
		Name:    name,
	}
	entry, err := s.moduleTable.Find(key)
	if err != nil {
		log.Println("invalid module for", domain+":"+project+":"+name, err)
		writeErrorWebSocket(ctx.Conn, "invalid module")
		return
	}

	if entry.BuildStatus == nil || entry.BuildStatus.Status == module.ModuleBuildPending {
		writeErrorWebSocket(ctx.Conn, "build not started")
		return
	}

	logsCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go func() {
		switch entry.BuildStatus.Status {
		case module.ModuleBuildCompleted, module.ModuleBuildFailed:
			size := len(entry.BuildStatus.Logs)
			for i := 0; i < size; i = i + 1024 {
				last := i + 1024
				if last > size {
					last = size
				}
				msg := entry.BuildStatus.Logs[i:last]
				err = ctx.Conn.WriteMessage(gorillaSocket.TextMessage, []byte(msg))
				if err != nil {
					log.Println("failed writing message to websocket", err)
					break
				}
			}
		case module.ModuleBuildInProgress:
			bName := "m-" + entry.Id.String()
			podLogOpts := corev1.PodLogOptions{Follow: true}
			req := podClient.GetLogs(bName, &podLogOpts)
			podLogs, err := req.Stream(logsCtx)
			if err != nil {
				log.Println("failed to open pod logs", err)
				return
			}
			defer podLogs.Close()
			for {
				buf := make([]byte, 1024)

				numBytes, err := podLogs.Read(buf)
				if numBytes == 0 {
					break
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Println("got error while reading pod logs", err)
					break
				}

				err = ctx.Conn.WriteMessage(gorillaSocket.TextMessage, buf[:numBytes])
				if err != nil {
					log.Println("failed writing message to websocket", err)
					break
				}
			}
		}
	}()

	for {
		_, _, err := ctx.Conn.ReadMessage()
		if err != nil {
			// we observed an error while reading from websocket
			// mostly happens due to connection close
			// ensure a closer of terminal by sending a SIGTERM
			// to the running child process
			break
		}
	}
	log.Println("module build logs websocket closed")
}

func init() {
	// get namespace of the pod in which it is running
	bytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalln("failed to get pod namespace ", err)
	}
	Namespace = string(bytes)

	k8sConfig := ctrl.GetConfigOrDie()

	// load kube client by config for compass controller
	clientset, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalln("failed to load k8s config for controller ", err)
	}

	podClient = clientset.CoreV1().Pods(Namespace)
}
