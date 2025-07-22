package server

import (
	"context"
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/coredgeio/compass/controller/pkg/auth"
	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	"github.com/coredgeio/compass/pkg/utils"

	api "github.com/coredgeio/workflow-manager/api/workflow"
	"github.com/coredgeio/workflow-manager/pkg/pattern"
	"github.com/coredgeio/workflow-manager/pkg/runtime/module"
)

var (
	DefaultProject string = "default-project"
	DefaultDomain         = "default-domain"
)

type ModuleApiServer struct {
	api.UnimplementedModuleApiServer
	moduleTbl *module.ModuleTable
}

func NewModuleApiServer() *ModuleApiServer {
	moduleTbl, err := module.LocateModuleTable()
	if err != nil {
		log.Fatalln("NewModuleApiServer: failed to locate module table", err)
	}
	return &ModuleApiServer{
		moduleTbl: moduleTbl,
	}
}

func moduleBuildStatusToApi(status module.ModuleBuildStatusType) api.ModuleBuildDef_Status {
	switch status {
	case module.ModuleBuildInProgress:
		return api.ModuleBuildDef_InProgress
	case module.ModuleBuildCompleted:
		return api.ModuleBuildDef_Completed
	case module.ModuleBuildFailed:
		return api.ModuleBuildDef_Failed
	}
	return api.ModuleBuildDef_Scheduled
}

func (s *ModuleApiServer) allocateIdentifier() (*uuid.UUID, error) {
	var err error
	for i := 1; i < 4; i++ {
		if i > 1 {
			log.Println("Failed to allocate base image id, retrying!!!")
		}
		uid := uuid.New()
		entry, _ := s.moduleTbl.FindById(&uid)
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

func (s *ModuleApiServer) ListModules(ctx context.Context, req *api.ModulesListReq) (*api.ModulesListResp, error) {
	count, err := s.moduleTbl.GetCountInProject(req.Domain, req.Project)
	if err != nil {
		log.Println("error fetching module count in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	resp := &api.ModulesListResp{
		Count: int32(count),
		Items: []*api.ModulesListEntry{},
	}
	list, err := s.moduleTbl.GetListInProject(req.Domain, req.Project, int64(req.Offset), int64(req.Limit))
	if err != nil {
		log.Println("error fetching module list in project", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	for _, entry := range list {
		var buildStatus module.ModuleBuildStatusType
		buildTime := int64(0)
		if entry.BuildStatus != nil {
			buildStatus = entry.BuildStatus.Status
			buildTime = entry.BuildStatus.BuildTime
		}
		item := &api.ModulesListEntry{
			Id:         entry.Id.String(),
			Name:       entry.Key.Name,
			Desc:       utils.PString(entry.Desc),
			CreatedBy:  entry.CreatedBy,
			CreateTime: entry.CreateTime,
			LastUpdate: entry.LastUpdate,
			Tags:       entry.Tags,
			IsDeleted:  entry.IsDeleted,
			InputKeys:  map[string]*api.InputKeyData{},
			OutputKeys: map[string]*api.OutputKeyData{},
			Build: &api.ModuleBuildStatus{
				Status:    moduleBuildStatusToApi(buildStatus),
				TimeStamp: buildTime,
			},
			Request: &api.ModuleRequestStatus{},
		}
		for k, v := range entry.InputKeys {
			item.InputKeys[k] = &api.InputKeyData{
				DataType:   v.DataType,
				Opt:        v.Opt,
				DefaultVal: v.DefaultVal,
			}
		}
		for k, v := range entry.OutputKeys {
			item.OutputKeys[k] = &api.OutputKeyData{
				DataType:  v.DataType,
				ValueFrom: v.ValueFrom,
			}
		}
		resp.Items = append(resp.Items, item)
	}

	return resp, nil
}

func (s *ModuleApiServer) CreateModule(ctx context.Context, req *api.ModuleCreateReq) (*api.ModuleCreateResp, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing mandatory parameters")
	}
	if !pattern.IsBasicName(req.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "%q invalid module name", req.Name)
	}
	userInfo, _ := auth.GetUserInfo(ctx)
	id, err := s.allocateIdentifier()
	if err != nil {
		log.Printf("unable to allocate UUID for module %v, error: %s", req, err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	entry := &module.ModuleEntry{
		Key: module.ModuleKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    req.Name,
		},
		Id:         id,
		Desc:       utils.StringP(req.Desc),
		CreatedBy:  userInfo.UserName,
		Tags:       req.Tags,
		InputKeys:  module.InputKeysType{},
		OutputKeys: module.OutputKeysType{},
	}
	for k, v := range req.InputKeys {
		entry.InputKeys[k] = &module.InputKeyData{
			DataType:   v.DataType,
			Opt:        v.Opt,
			DefaultVal: v.DefaultVal,
		}
	}
	for k, v := range req.OutputKeys {
		entry.OutputKeys[k] = &module.OutputKeyData{
			DataType:  v.DataType,
			ValueFrom: v.ValueFrom,
		}
	}
	if req.BuildConfig == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Missing build config")
	}
	entry.BuildConfig = &module.ModuleBuildConfig{
		BaseImage:   req.BuildConfig.BaseImage,
		BuildScript: req.BuildConfig.BuildScript,
		Env:         req.BuildConfig.Env,
		EntryPoint:  req.BuildConfig.EntryPoint,
	}
	for _, v := range req.BuildConfig.Files {
		info := &module.ModuleFileInfo{
			Name:    v.Name,
			Content: []byte(v.Content),
			Perm:    v.Perm,
		}
		entry.BuildConfig.Files = append(entry.BuildConfig.Files, info)
	}

	if req.BuildConfig.GitInfo != nil {
		if req.BuildConfig.GitInfo.Url == "" &&
			(req.BuildConfig.GitInfo.GitRef != "" ||
				req.BuildConfig.GitInfo.WorkingDir != "") {
			return nil, status.Errorf(codes.InvalidArgument, "Missing git repo url")
		}
		entry.BuildConfig.GitInfo = &module.ModuleGitInfo{
			Url:        req.BuildConfig.GitInfo.Url,
			GitRef:     req.BuildConfig.GitInfo.GitRef,
			WorkingDir: req.BuildConfig.GitInfo.WorkingDir,
		}
	}

	err = s.moduleTbl.Add(entry)
	if err != nil {
		if pkgerrors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.AlreadyExists, "Entry %q already exists", entry.Key)
		}
		log.Println("Error creating module", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}

	return &api.ModuleCreateResp{}, nil
}

func (s *ModuleApiServer) GetModule(ctx context.Context, req *api.ModuleEntryKey) (*api.ModuleGetResp, error) {
	key := &module.ModuleKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}
	entry, err := s.moduleTbl.Find(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error finding module", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	var buildStatus module.ModuleBuildStatusType
	buildTime := int64(0)
	if entry.BuildStatus != nil {
		buildStatus = entry.BuildStatus.Status
		buildTime = entry.BuildStatus.BuildTime
	}
	resp := &api.ModuleGetResp{
		Id:         entry.Id.String(),
		Name:       entry.Key.Name,
		Desc:       utils.PString(entry.Desc),
		CreatedBy:  entry.CreatedBy,
		CreateTime: entry.CreateTime,
		LastUpdate: entry.LastUpdate,
		Tags:       entry.Tags,
		InputKeys:  map[string]*api.InputKeyData{},
		OutputKeys: map[string]*api.OutputKeyData{},
		IsDeleted:  entry.IsDeleted,
		Build: &api.ModuleBuildStatus{
			Status:    moduleBuildStatusToApi(buildStatus),
			TimeStamp: buildTime,
		},
		Request: &api.ModuleRequestStatus{},
	}
	for k, v := range entry.InputKeys {
		resp.InputKeys[k] = &api.InputKeyData{
			DataType:   v.DataType,
			Opt:        v.Opt,
			DefaultVal: v.DefaultVal,
		}
	}
	for k, v := range entry.OutputKeys {
		resp.OutputKeys[k] = &api.OutputKeyData{
			DataType:  v.DataType,
			ValueFrom: v.ValueFrom,
		}
	}
	if entry.BuildConfig != nil {
		resp.BuildConfig = &api.ModuleBuildConfig{
			BaseImage:   entry.BuildConfig.BaseImage,
			BuildScript: entry.BuildConfig.BuildScript,
			Env:         entry.BuildConfig.Env,
			EntryPoint:  entry.BuildConfig.EntryPoint,
		}
		for _, v := range entry.BuildConfig.Files {
			info := &api.ModuleFileInfo{
				Name:    v.Name,
				Content: string(v.Content),
				Perm:    v.Perm,
			}
			resp.BuildConfig.Files = append(resp.BuildConfig.Files, info)
		}
		if entry.BuildConfig.GitInfo != nil {
			resp.BuildConfig.GitInfo = &api.ModuleGitInfo{
				Url:        entry.BuildConfig.GitInfo.Url,
				GitRef:     entry.BuildConfig.GitInfo.GitRef,
				WorkingDir: entry.BuildConfig.GitInfo.WorkingDir,
			}
		}
	}
	return resp, nil
}

func (s *ModuleApiServer) UpdateModule(ctx context.Context, req *api.ModuleCreateReq) (*api.ModuleCreateResp, error) {
	// Track whether the input/output keys are being cleared
	clearInputKeys := len(req.InputKeys) == 0
	clearOutputKeys := len(req.OutputKeys) == 0

	key := module.ModuleKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}
	entry := &module.ModuleEntry{
		Key:        key,
		Desc:       utils.StringP(req.Desc),
		Tags:       req.Tags,
		InputKeys:  module.InputKeysType{},
		OutputKeys: module.OutputKeysType{},
	}
	for k, v := range req.InputKeys {
		clearInputKeys = false // Not empty anymore
		entry.InputKeys[k] = &module.InputKeyData{
			DataType:   v.DataType,
			Opt:        v.Opt,
			DefaultVal: v.DefaultVal,
		}
	}
	for k, v := range req.OutputKeys {
		clearOutputKeys = false // Not empty anymore
		entry.OutputKeys[k] = &module.OutputKeyData{
			DataType:  v.DataType,
			ValueFrom: v.ValueFrom,
		}
	}
	if req.BuildConfig == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Missing build config")
	}
	entry.BuildConfig = &module.ModuleBuildConfig{
		BaseImage:   req.BuildConfig.BaseImage,
		BuildScript: req.BuildConfig.BuildScript,
		Env:         req.BuildConfig.Env,
		EntryPoint:  req.BuildConfig.EntryPoint,
	}
	for _, v := range req.BuildConfig.Files {
		info := &module.ModuleFileInfo{
			Name:    v.Name,
			Content: []byte(v.Content),
			Perm:    v.Perm,
		}
		entry.BuildConfig.Files = append(entry.BuildConfig.Files, info)
	}

	if req.BuildConfig.GitInfo != nil {
		if req.BuildConfig.GitInfo.Url == "" &&
			(req.BuildConfig.GitInfo.GitRef != "" ||
				req.BuildConfig.GitInfo.WorkingDir != "") {
			return nil, status.Errorf(codes.InvalidArgument, "Missing git repo url")
		}
		entry.BuildConfig.GitInfo = &module.ModuleGitInfo{
			Url:        req.BuildConfig.GitInfo.Url,
			GitRef:     req.BuildConfig.GitInfo.GitRef,
			WorkingDir: req.BuildConfig.GitInfo.WorkingDir,
		}
	}

	err := s.moduleTbl.Update(entry)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
		}
		log.Println("Error updating module", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	// Only explicitly clear keys if the user sent zero keys and we didn't set any during this request
	if clearInputKeys {
		err := s.moduleTbl.EmptyInputKeys(&key)
		if err != nil {
			if pkgerrors.IsNotFound(err) {
				return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
			}
			log.Println("Error updating module", entry.Key, "error", err)
			return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
		}
	}
	if clearOutputKeys {
		err := s.moduleTbl.EmptyOutputKeys(&key)
		if err != nil {
			if pkgerrors.IsNotFound(err) {
				return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
			}
			log.Println("Error updating module", entry.Key, "error", err)
			return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
		}
	}
	return &api.ModuleCreateResp{}, nil
}

func (s *ModuleApiServer) DeleteModule(ctx context.Context, req *api.ModuleEntryKey) (*api.ModuleDeleteResp, error) {
	entry := &module.ModuleEntry{
		Key: module.ModuleKey{
			Domain:  req.Domain,
			Project: req.Project,
			Name:    req.Name,
		},
		IsDeleted: true,
	}
	err := s.moduleTbl.Update(entry)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", entry.Key)
		}
		log.Println("Error deleting module", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	resp := &api.ModuleDeleteResp{}
	return resp, nil
}

func (s *ModuleApiServer) RebuildModule(ctx context.Context, req *api.ModuleEntryKey) (*api.ModuleRebuildResp, error) {
	key := &module.ModuleKey{
		Domain:  req.Domain,
		Project: req.Project,
		Name:    req.Name,
	}
	err := s.moduleTbl.ResetBuildStatus(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", *key)
		}
		log.Println("Error triggering rebuild for module", *key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	resp := &api.ModuleRebuildResp{}
	return resp, nil
}

func (s *ModuleApiServer) CreateCatalogRequest(ctx context.Context, req *api.ModuleCatalogCreateReq) (*api.ModuleCatalogCreateResp, error) {
	log.Println("Create Catalog", req)
	resp := &api.ModuleCatalogCreateResp{}
	return resp, nil
}

func (s *ModuleApiServer) DeleteCatalogRequest(ctx context.Context, req *api.ModuleCatalogDeleteReq) (*api.ModuleCatalogDeleteResp, error) {
	log.Println("Delete Catalog", req)
	resp := &api.ModuleCatalogDeleteResp{}
	return resp, nil
}

func (s *ModuleApiServer) AddModuleComment(ctx context.Context, req *api.ModuleCommentAddReq) (*api.ModuleCommentAddResp, error) {
	log.Println("Add module comment", req)
	return &api.ModuleCommentAddResp{}, nil
}

func (s *ModuleApiServer) ListModuleCatalog(ctx context.Context, req *api.ModuleCatalogListReq) (*api.ModuleCatalogListResp, error) {
	resp := &api.ModuleCatalogListResp{
		Count: int32(3),
		Items: []*api.ModuleCatalogListEntry{
			{
				Id:         uuid.New().String(),
				Name:       "catalog-module-1",
				Desc:       "catalog test module 1",
				CreatedBy:  "prabhjot@coredge.io",
				CreateTime: int64(1727379128),
				LastUpdate: int64(1727380608),
				Tags:       []string{"fwaas", "dc"},
				InputKeys: map[string]*api.InputKeyData{
					"in1": {
						DataType: "string",
						Opt:      false,
					},
					"in2": {
						DataType: "string",
						Opt:      true,
					},
				},
				OutputKeys: map[string]*api.OutputKeyData{
					"out1": {
						DataType: "string",
					},
					"out2": {
						DataType: "string",
					},
				},
				Image:         "docker.io/coredgeio/catalog-module-1:2.1",
				LatestVersion: "2.1",
				IsArchived:    false,
				IsDeleted:     false,
			},
			{
				Id:         uuid.New().String(),
				Name:       "catalog-module-2",
				Desc:       "Catalog test module 2",
				CreatedBy:  "prabhjot@coredge.io",
				CreateTime: int64(1727380091),
				LastUpdate: int64(1727380918),
				Tags:       []string{"lbaas", "demo"},
				InputKeys: map[string]*api.InputKeyData{
					"in1": {
						DataType: "string",
						Opt:      true,
					},
					"in2": {
						DataType: "string",
						Opt:      true,
					},
				},
				OutputKeys: map[string]*api.OutputKeyData{
					"out1": {
						DataType: "string",
					},
				},
				Image:         "docker.io/coredgeio/catalog-module-2:5.1",
				LatestVersion: "5.1",
				IsArchived:    false,
				IsDeleted:     false,
			},
		},
	}
	return resp, nil
}

func (s *ModuleApiServer) ListModuleCatalogVer(ctx context.Context, req *api.ModuleCatalogVerListReq) (*api.ModuleCatalogVerListResp, error) {
	resp := &api.ModuleCatalogVerListResp{
		Count: int32(3),
		Items: []*api.ModuleCatalogVerListEntry{
			{
				Name:       "catalog-module-1",
				Version:    "2.1",
				Desc:       "catalog test module 1",
				CreatedBy:  "prabhjot@coredge.io",
				CreateTime: int64(1727379128),
				LastUpdate: int64(1727380608),
				Tags:       []string{"fwaas", "dc"},
				InputKeys: map[string]*api.InputKeyData{
					"in1": {
						DataType: "string",
						Opt:      false,
					},
					"in2": {
						DataType: "string",
						Opt:      true,
					},
				},
				OutputKeys: map[string]*api.OutputKeyData{
					"out1": {
						DataType: "string",
					},
					"out2": {
						DataType: "string",
					},
				},
				Image:      "docker.io/coredgeio/catalog-module-1:2.1",
				IsLatest:   true,
				IsArchived: false,
				IsDeleted:  false,
			},
			{
				Name:       "catalog-module-1",
				Version:    "2.0",
				Desc:       "Catalog test module 1",
				CreatedBy:  "prabhjot@coredge.io",
				CreateTime: int64(1727380091),
				LastUpdate: int64(1727380918),
				Tags:       []string{"lbaas", "demo"},
				InputKeys: map[string]*api.InputKeyData{
					"in1": {
						DataType: "string",
						Opt:      true,
					},
					"in2": {
						DataType: "string",
						Opt:      true,
					},
				},
				OutputKeys: map[string]*api.OutputKeyData{
					"out1": {
						DataType: "string",
					},
				},
				Image:      "docker.io/coredgeio/catalog-module-1:2.0",
				IsLatest:   false,
				IsArchived: false,
				IsDeleted:  false,
			},
		},
	}
	return resp, nil
}

func (s *ModuleApiServer) ListModuleCatalogRequest(ctx context.Context, req *api.ModuleCatalogRequestListReq) (*api.ModuleCatalogRequestListResp, error) {
	resp := &api.ModuleCatalogRequestListResp{
		Count: int32(2),
		Items: []*api.ModuleCatalogRequestListEntry{
			{
				Id:     uuid.New().String(),
				Status: api.ModuleRequestStatus_Submitted,
				Module: &api.ModuleRequestModuleInfo{
					Domain:  "default",
					Project: "demo",
					Name:    "test-module-1",
					Id:      uuid.New().String(),
				},
				Catalog: &api.ModuleRequestCatalogInfo{
					Id:   uuid.New().String(),
					Name: "catalog-module-1",
					Desc: "Catalog Module entry 1",
				},
			},
			{
				Id:     uuid.New().String(),
				Status: api.ModuleRequestStatus_Submitted,
				Module: &api.ModuleRequestModuleInfo{
					Domain:  "default",
					Project: "demo",
					Name:    "test-module-2",
					Id:      uuid.New().String(),
				},
				Catalog: &api.ModuleRequestCatalogInfo{
					Id:   uuid.New().String(),
					Name: "catalog-module-2",
					Desc: "Catalog Module entry 2",
				},
			},
		},
	}
	return resp, nil
}
