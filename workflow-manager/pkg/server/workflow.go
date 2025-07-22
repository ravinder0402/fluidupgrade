package server

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	"github.com/coredgeio/compass/pkg/utils"

	api "github.com/coredgeio/workflow-manager/api/workflow"
	wRuntime "github.com/coredgeio/workflow-manager/pkg/runtime/workflow"
)

type WorkflowApiServer struct {
	api.UnimplementedWorkflowApiServer
	workflowTbl *wRuntime.WorkflowTable
}

func NewWorkflowApiServer() *WorkflowApiServer {
	workflowTbl, err := wRuntime.LocateWorkflowTable()
	if err != nil {
		log.Fatalln("WorkflowApiServer: failed to locate workflow table", err)
	}
	return &WorkflowApiServer{
		workflowTbl: workflowTbl,
	}
}

func (s *WorkflowApiServer) nodeTypeRuntimeToApi(t wRuntime.WorkflowNodeType) api.WorkflowNode_NodeType {
	switch t {
	case wRuntime.CatalogNode:
		return api.WorkflowNode_Catalog
	case wRuntime.UserInputNode:
		return api.WorkflowNode_UserInput
	}
	return api.WorkflowNode_Module
}

func (s *WorkflowApiServer) workflowStateRuntimetoApi(state wRuntime.WorkflowState) api.WorkflowDef_Status {
	switch state {
	case wRuntime.WorkflowScheduled:
		return api.WorkflowDef_Scheduled
	case wRuntime.WorkflowRunning:
		return api.WorkflowDef_Running
	case wRuntime.WorkflowCompleted:
		return api.WorkflowDef_Completed
	case wRuntime.WorkflowFailed:
		return api.WorkflowDef_Failed
	}
	return api.WorkflowDef_Created
}

func (s *WorkflowApiServer) ListWorkflows(ctx context.Context, req *api.WorkflowListReq) (*api.WorkflowListResp, error) {
	count, err := s.workflowTbl.GetCountInProject(req.Domain, req.Project)
	if err != nil {
		log.Println("error fetching workflow count in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	resp := &api.WorkflowListResp{
		Count: int32(count),
		Items: []*api.WorkflowListEntry{},
	}
	list, err := s.workflowTbl.GetListInProject(req.Domain, req.Project, int64(req.Offset), int64(req.Limit))
	if err != nil {
		log.Println("error fetching workflow list in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	for _, entry := range list {
		state := wRuntime.WorkflowCreated
		if entry.Status != nil {
			state = entry.Status.State
		}
		item := &api.WorkflowListEntry{
			Name:       entry.Key.Name,
			Template:   entry.Template,
			Desc:       utils.PString(entry.Desc),
			Status:     s.workflowStateRuntimetoApi(state),
			CreatedBy:  entry.CreatedBy,
			CreateTime: entry.CreateTime,
			StartTime:  entry.StartTime,
			EndTime:    entry.EndTime,
			Tags:       entry.Tags,
			IsDeleted:  entry.IsDeleted,
		}
		resp.Items = append(resp.Items, item)
	}

	return resp, nil
}

func (s *WorkflowApiServer) DeleteWorkflow(ctx context.Context, req *api.WorkflowDeleteReq) (*api.WorkflowDeleteResp, error) {
	entry := &wRuntime.WorkflowEntry{
		Key: wRuntime.WorkflowKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    req.Name,
		},
		IsDeleted: true,
	}

	err := s.workflowTbl.Update(entry)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
		}
		log.Println("Error deleting workflow", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	return &api.WorkflowDeleteResp{}, nil
}

func (s *WorkflowApiServer) GetWorkflow(ctx context.Context, req *api.WorkflowGetReq) (*api.WorkflowGetResp, error) {
	key := &wRuntime.WorkflowKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}

	entry, err := s.workflowTbl.Find(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error finding workflow", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	state := wRuntime.WorkflowCreated
	if entry.Status != nil {
		state = entry.Status.State
	}
	resp := &api.WorkflowGetResp{
		Name:       entry.Key.Name,
		Template:   entry.Template,
		Desc:       utils.PString(entry.Desc),
		Status:     s.workflowStateRuntimetoApi(state),
		CreatedBy:  entry.CreatedBy,
		CreateTime: entry.CreateTime,
		StartTime:  entry.StartTime,
		EndTime:    entry.EndTime,
		Tags:       entry.Tags,
		IsDeleted:  entry.IsDeleted,
		Nodes:      []*api.WorkflowNode{},
		Links:      []*api.TemplateLink{},
	}

	if entry.Status != nil {
		for _, n := range entry.Status.Nodes {
			node := &api.WorkflowNode{
				Type:     s.nodeTypeRuntimeToApi(n.Type),
				Name:     n.Name,
				NodeId:   n.NodeId,
				ModuleId: n.ModuleId,
				X:        n.X,
				Y:        n.Y,
			}
			if n.Type == wRuntime.UserInputNode {
				node.Status = api.WorkflowDef_Completed
			} else {
				node.Status = s.workflowStateRuntimetoApi(n.State)
				if n.State == wRuntime.WorkflowFailed {
					node.Err = n.Error
				}
				for _, data := range n.Inputs {
					in := &api.WorkflowNodeData{
						Name:  data.Name,
						Value: data.Value,
					}
					node.Inputs = append(node.Inputs, in)
				}
				for _, data := range n.Outputs {
					out := &api.WorkflowNodeData{
						Name:  data.Name,
						Value: data.Value,
					}
					node.Outputs = append(node.Outputs, out)
				}
			}
			resp.Nodes = append(resp.Nodes, node)
		}

		for _, l := range entry.Status.Links {
			link := &api.TemplateLink{
				Source:    l.Source,
				SourceVar: l.SourceVar,
				Target:    l.Target,
				TargetVar: l.TargetVar,
			}
			resp.Links = append(resp.Links, link)
		}
	}
	return resp, nil
}
