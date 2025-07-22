package websocket

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/gorilla/mux"
	gorillaSocket "github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/coredgeio/compass/pkg/infra/websocket"

	"github.com/coredgeio/workflow-manager/pkg/runtime/workflow"
)

type WorkflowLogsServer struct {
	websocket.Handler
	workflowTable *workflow.WorkflowTable
}

func CreateWorkflowLogsServer(router *mux.Router) {
	workflowTable, err := workflow.LocateWorkflowTable()
	if err != nil {
		log.Fatalln("WorkflowLogsServer failed to locate workflow runtime table", err)
	}

	server := &WorkflowLogsServer{
		workflowTable: workflowTable,
	}
	ws := websocket.AllocateServer(server)

	v1Router := router.PathPrefix("/ws/workflow/v1").Subrouter()
	v1Router.HandleFunc("/domain/{domain}/project/{project}/workflow/{name}/node/{id}/logs", ws.ServeHTTP).Methods("GET")
}

func (s *WorkflowLogsServer) ServeConnection(ctx *websocket.Context) {
	domain, ok := ctx.Vars["domain"]
	if !ok {
		log.Println("got invalid domain for workflow execution log access")
		writeErrorWebSocket(ctx.Conn, "invalid domain")
		return
	}

	project, ok := ctx.Vars["project"]
	if !ok {
		log.Println("got invalid project for workflow execution log access")
		writeErrorWebSocket(ctx.Conn, "invalid project")
		return
	}

	name, ok := ctx.Vars["name"]
	if !ok {
		log.Println("got invalid workflow name for execution log access")
		writeErrorWebSocket(ctx.Conn, "invalid workflow name")
		return
	}

	nodeId, ok := ctx.Vars["id"]
	if !ok {
		log.Println("got invalid workflow node for execution log access")
		writeErrorWebSocket(ctx.Conn, "invalid workflow node id")
		return
	}

	key := &workflow.WorkflowKey{
		Domain:  domain,
		Project: project,
		Name:    name,
	}
	entry, err := s.workflowTable.Find(key)
	if err != nil {
		log.Println("invalid workflow for", domain+":"+project+":"+name, err)
		writeErrorWebSocket(ctx.Conn, "invalid workflow")
		return
	}

	if entry.Status == nil || entry.Status.State == workflow.WorkflowCreated ||
		entry.Status.State == workflow.WorkflowScheduled {
		writeErrorWebSocket(ctx.Conn, "execution not yet started")
		return
	}

	argoId := ""
	var state workflow.WorkflowState
	for _, n := range entry.Status.Nodes {
		if n.NodeId == nodeId {
			argoId = n.ArgoId
			state = n.State
			break
		}
	}

	if argoId == "" {
		writeErrorWebSocket(ctx.Conn, "execution not yet started")
		return
	}

	logsCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go func() {
		switch state {
		case workflow.WorkflowCompleted, workflow.WorkflowFailed, workflow.WorkflowRunning:
			podList, err := podClient.List(context.TODO(),
				metav1.ListOptions{
					LabelSelector: fmt.Sprintf("workflows.argoproj.io/workflow=%s", entry.Key.Name),
				})
			if err != nil {
				log.Println("failed to get list of pod for workflow execution")
				return
			}
			podName := ""
			for _, pod := range podList.Items {
				if value, ok := pod.Annotations["workflows.argoproj.io/node-id"]; ok && value == argoId {
					podName = pod.Name
					break
				}
			}
			if podName == "" {
				log.Println("failed to get pod name, workflow execution")
				return
			}
			podLogOpts := corev1.PodLogOptions{Follow: true, Container: "main"}
			req := podClient.GetLogs(podName, &podLogOpts)
			podLogs, err := req.Stream(logsCtx)
			if err != nil {
				log.Println("failed to open pod workflow execution logs", err)
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
					log.Println("got error while reading pod workflow execution logs", err)
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
	log.Println("workflow execution logs websocket closed")
}
