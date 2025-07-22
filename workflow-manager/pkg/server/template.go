package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/coredgeio/compass/controller/pkg/auth"
	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	"github.com/coredgeio/compass/pkg/utils"

	api "github.com/coredgeio/workflow-manager/api/workflow"
	"github.com/coredgeio/workflow-manager/pkg/pattern"
	"github.com/coredgeio/workflow-manager/pkg/runtime/template"
	wRuntime "github.com/coredgeio/workflow-manager/pkg/runtime/workflow"
)

type WorkflowTemplateApiServer struct {
	api.UnimplementedWorkflowTemplateApiServer
	templateTbl *template.TemplateTable
	workflowTbl *wRuntime.WorkflowTable
}

func NewWorkflowTemplateApiServer() *WorkflowTemplateApiServer {
	templateTbl, err := template.LocateTemplateTable()
	if err != nil {
		log.Fatalln("WorkflowTemplateApiServer: failed to locate template table", err)
	}
	workflowTbl, err := wRuntime.LocateWorkflowTable()
	if err != nil {
		log.Fatalln("WorkflowTemplateApiServer: failed to locate workflow table", err)
	}
	return &WorkflowTemplateApiServer{
		templateTbl: templateTbl,
		workflowTbl: workflowTbl,
	}
}

func nodeTypeApiToRuntime(t api.TemplateNode_NodeType) template.TemplateNodeType {
	switch t {
	case api.TemplateNode_Catalog:
		return template.CatalogNode
	case api.TemplateNode_UserInput:
		return template.UserInputNode
	}
	return template.ModuleNode
}

func nodeTypeRuntimeToApi(t template.TemplateNodeType) api.TemplateNode_NodeType {
	switch t {
	case template.CatalogNode:
		return api.TemplateNode_Catalog
	case template.UserInputNode:
		return api.TemplateNode_UserInput
	}
	return api.TemplateNode_Module
}

type stageNodeMap map[string]*stageNode

type stageNode struct {
	stage    int
	module   bool
	nodeInfo *template.TemplateNode
	waitFor  stageNodeMap
	signal   stageNodeMap
}

func getStageInfo(stage, max int, start *stageNode) (bool, error) {
	if max < 0 {
		return false, pkgerrors.Wrap(pkgerrors.InvalidArgument, "Bad cyclic dependency in template")
	}
	for _, v := range start.waitFor {
		if v.stage == -1 {
			// node is not processed yet
			// return from here without error
			return false, nil
		}
		if v.stage >= stage {
			stage = v.stage + 1
		}
	}
	start.stage = stage
	start.nodeInfo.Stage = int32(stage)
	for _, v := range start.signal {
		_, err := getStageInfo((stage + 1), (max - 1), v)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func validateAndBuildStages(nodes []*template.TemplateNode, links []*template.TemplateLink) error {
	sMap := stageNodeMap{}
	moduleMap := stageNodeMap{}
	// populate map keys and check nodes duplication
	for _, n := range nodes {
		_, ok := sMap[n.NodeId]
		if ok {
			// duplicate node
			return pkgerrors.Wrap(pkgerrors.InvalidArgument, "duplicate node")
		}
		// TODO(prabhjot) we should also check module availability
		sNode := &stageNode{
			stage:    -1,
			nodeInfo: n,
			waitFor:  stageNodeMap{},
			signal:   stageNodeMap{},
		}
		sMap[n.NodeId] = sNode
		switch n.Type {
		case template.ModuleNode, template.CatalogNode:
			moduleMap[n.NodeId] = sNode
			sNode.module = true
		}
	}

	for _, l := range links {
		// TODO(prabhjot) need to identify way to check links duplicity
		source, ok := sMap[l.Source]
		if !ok {
			return pkgerrors.Wrap(pkgerrors.InvalidArgument, "invalid link")
		}
		target, ok := sMap[l.Target]
		if !ok {
			return pkgerrors.Wrap(pkgerrors.InvalidArgument, "invalid link")
		}
		if source.module && target.module {
			source.signal[l.Target] = target
			target.waitFor[l.Source] = source
		}
	}

	var start *stageNode
	for _, v := range moduleMap {
		if len(v.waitFor) == 0 {
			if start != nil {
				return pkgerrors.Wrap(pkgerrors.InvalidArgument, "template includes more than one starting point")
			}
			start = v
		}
	}
	if start == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "template has no starting point")
	}

	_, err := getStageInfo(0, len(moduleMap), start)

	if err != nil {
		return err
	}

	for _, v := range moduleMap {
		if v.stage == -1 {
			return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Bad or cyclic dependency in template")
		}
	}

	return err
}

func (s *WorkflowTemplateApiServer) allocateIdentifier() (*uuid.UUID, error) {
	var err error
	for i := 1; i < 4; i++ {
		if i > 1 {
			log.Println("Failed to allocate base image id, retrying!!!")
		}
		uid := uuid.New()
		entry, _ := s.templateTbl.FindById(&uid)
		if entry != nil {
			err = pkgerrors.Wrap(pkgerrors.Unknown, "failed to allocate identifier, found an existing namespace whose UUID conflicts with the generated UUID")
			// retry if we have attempts left
			continue
		}
		// uuid not in use we are good to use it
		return &uid, nil
	}
	return nil, err
}

func (s *WorkflowTemplateApiServer) ListTemplates(ctx context.Context, req *api.TemplateListReq) (*api.TemplateListResp, error) {
	count, err := s.templateTbl.GetCountInProject(req.Domain, req.Project)
	if err != nil {
		log.Println("error fetching template count in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	resp := &api.TemplateListResp{
		Count: int32(count),
		Items: []*api.TemplateListEntry{},
	}
	list, err := s.templateTbl.GetListInProject(req.Domain, req.Project, int64(req.Offset), int64(req.Limit))
	if err != nil {
		log.Println("error fetching template list in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	for _, entry := range list {
		item := &api.TemplateListEntry{
			Name:       entry.Key.Name,
			Id:         entry.Id.String(),
			Desc:       utils.PString(entry.Desc),
			CreatedBy:  entry.CreatedBy,
			CreateTime: entry.CreateTime,
			LastUpdate: entry.LastUpdate,
			Tags:       entry.Tags,
			IsDeleted:  entry.IsDeleted,
		}
		resp.Items = append(resp.Items, item)
	}

	return resp, nil
}

func (s *WorkflowTemplateApiServer) CreateTemplate(ctx context.Context, req *api.TemplateCreateReq) (*api.TemplateCreateResp, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing mandatory parameters")
	}
	if !pattern.IsBasicName(req.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "%q invalid template name", req.Name)
	}
	userInfo, _ := auth.GetUserInfo(ctx)
	id, err := s.allocateIdentifier()
	if err != nil {
		log.Printf("unable to allocate UUID for template %v, error: %s", req, err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	entry := &template.TemplateEntry{
		Key: template.TemplateKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    req.Name,
		},
		Id:        id,
		Desc:      utils.StringP(req.Desc),
		CreatedBy: userInfo.UserName,
		Tags:      req.Tags,
		Nodes:     []*template.TemplateNode{},
		Links:     []*template.TemplateLink{},
	}
	for _, n := range req.Nodes {
		node := &template.TemplateNode{
			Type:     nodeTypeApiToRuntime(n.Type),
			Name:     n.Name,
			NodeId:   n.NodeId,
			ModuleId: n.ModuleId,
			X:        n.X,
			Y:        n.Y,
		}
		if n.Type == api.TemplateNode_UserInput {
			// ensure that the user input data exists
			if n.UserData == nil {
				return nil, status.Errorf(codes.InvalidArgument, "%q: missing user data for user input node", n.NodeId)
			}
			node.UserData = &template.TemplateUserInputNodeData{
				Name:       n.UserData.Name,
				Desc:       n.UserData.Desc,
				DefaultVal: n.UserData.DefaultVal,
				Opt:        n.UserData.Opt,
			}
		}
		entry.Nodes = append(entry.Nodes, node)
	}
	for _, l := range req.Links {
		link := &template.TemplateLink{
			Source:    l.Source,
			SourceVar: l.SourceVar,
			Target:    l.Target,
			TargetVar: l.TargetVar,
		}
		entry.Links = append(entry.Links, link)
	}

	err = validateAndBuildStages(entry.Nodes, entry.Links)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = s.templateTbl.Add(entry)
	if err != nil {
		if pkgerrors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.AlreadyExists, "Entry %q already exists", entry.Key)
		}
		log.Println("Error creating template", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	return &api.TemplateCreateResp{}, nil
}

func (s *WorkflowTemplateApiServer) GetTemplate(ctx context.Context, req *api.TemplateGetReq) (*api.TemplateGetResp, error) {
	key := &template.TemplateKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}
	entry, err := s.templateTbl.Find(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error finding template", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	resp := &api.TemplateGetResp{
		Name:       entry.Key.Name,
		Id:         entry.Id.String(),
		Desc:       utils.PString(entry.Desc),
		CreatedBy:  entry.CreatedBy,
		CreateTime: entry.CreateTime,
		LastUpdate: entry.LastUpdate,
		Tags:       entry.Tags,
		IsDeleted:  entry.IsDeleted,
		Nodes:      []*api.TemplateNode{},
		Links:      []*api.TemplateLink{},
	}

	for _, n := range entry.Nodes {
		node := &api.TemplateNode{
			Type:     nodeTypeRuntimeToApi(n.Type),
			Name:     n.Name,
			NodeId:   n.NodeId,
			ModuleId: n.ModuleId,
			X:        n.X,
			Y:        n.Y,
		}
		if n.Type == template.UserInputNode {
			if n.UserData == nil {
				// skip this entry as invalid
				continue
			}
			node.UserData = &api.TemplateUserInputNodeData{
				Name:       n.UserData.Name,
				Desc:       n.UserData.Desc,
				DefaultVal: n.UserData.DefaultVal,
				Opt:        n.UserData.Opt,
			}
		}
		resp.Nodes = append(resp.Nodes, node)
	}
	for _, l := range entry.Links {
		link := &api.TemplateLink{
			Source:    l.Source,
			SourceVar: l.SourceVar,
			Target:    l.Target,
			TargetVar: l.TargetVar,
		}
		resp.Links = append(resp.Links, link)
	}
	return resp, nil
}

func (s *WorkflowTemplateApiServer) UpdateTemplate(ctx context.Context, req *api.TemplateUpdateReq) (*api.TemplateUpdateResp, error) {
	entry := &template.TemplateEntry{
		Key: template.TemplateKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    req.Name,
		},
		Desc:  utils.StringP(req.Desc),
		Tags:  req.Tags,
		Nodes: []*template.TemplateNode{},
		Links: []*template.TemplateLink{},
	}
	for _, n := range req.Nodes {
		node := &template.TemplateNode{
			Type:     nodeTypeApiToRuntime(n.Type),
			Name:     n.Name,
			NodeId:   n.NodeId,
			ModuleId: n.ModuleId,
			X:        n.X,
			Y:        n.Y,
		}
		if n.Type == api.TemplateNode_UserInput {
			// ensure that the user input data exists
			if n.UserData == nil {
				return nil, status.Errorf(codes.InvalidArgument, "%q: missing user data for user input node", n.NodeId)
			}
			node.UserData = &template.TemplateUserInputNodeData{
				Name:       n.UserData.Name,
				Desc:       n.UserData.Desc,
				DefaultVal: n.UserData.DefaultVal,
				Opt:        n.UserData.Opt,
			}
		}
		entry.Nodes = append(entry.Nodes, node)
	}
	for _, l := range req.Links {
		link := &template.TemplateLink{
			Source:    l.Source,
			SourceVar: l.SourceVar,
			Target:    l.Target,
			TargetVar: l.TargetVar,
		}
		entry.Links = append(entry.Links, link)
	}

	err := validateAndBuildStages(entry.Nodes, entry.Links)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = s.templateTbl.Update(entry)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
		}
		log.Println("Error update template", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	return &api.TemplateUpdateResp{}, nil
}

func (s *WorkflowTemplateApiServer) DeleteTemplate(ctx context.Context, req *api.TemplateDeleteReq) (*api.TemplateDeleteResp, error) {
	key := &template.TemplateKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}

	err := s.templateTbl.Remove(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error deleting template", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	return &api.TemplateDeleteResp{}, nil
}

func templateToWorkflowName(name string) string {
	size := len(name)
	if size > 15 {
		size = 15
	}
	str := name[:size]
	if str[size-1] != '-' {
		str += "-"
	}
	str += fmt.Sprintf("%d", time.Now().UnixNano())
	return str
}

func (s *WorkflowTemplateApiServer) ExecuteTemplate(ctx context.Context, req *api.TemplateExecuteReq) (*api.TemplateExecuteResp, error) {
	key := &template.TemplateKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}
	entry, err := s.templateTbl.Find(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error finding template", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	userInfo, _ := auth.GetUserInfo(ctx)

	workflow := &wRuntime.WorkflowEntry{
		Key: wRuntime.WorkflowKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    templateToWorkflowName(req.Name),
		},
		Desc:      entry.Desc,
		Template:  req.Name,
		Inputs:    req.Inputs,
		CreatedBy: userInfo.UserName,
		Tags:      entry.Tags,
	}

	err = s.workflowTbl.Add(workflow)
	if err != nil {
		log.Println("Error creating workflow", workflow.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	return &api.TemplateExecuteResp{
		Domain:  workflow.Key.Domain,
		Project: workflow.Key.Project,
		Name:    workflow.Key.Name,
	}, nil
}
