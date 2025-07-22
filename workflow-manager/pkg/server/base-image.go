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
	"github.com/coredgeio/workflow-manager/pkg/runtime/baseimage"
)

type BaseImageApiServer struct {
	api.UnimplementedBaseImageApiServer
	imgTable *baseimage.BaseImageVersionTable
}

func NewBaseImageApiServer() *BaseImageApiServer {
	imgTable, err := baseimage.LocateBaseImageVersionTable()
	if err != nil {
		log.Fatalln("NewBaseImageApiServer: failed to locate BaseImageVersionTable", err)
	}
	return &BaseImageApiServer{
		imgTable: imgTable,
	}
}

func (s *BaseImageApiServer) allocateIdentifier() (*uuid.UUID, error) {
	var err error
	for i := 1; i < 4; i++ {
		if i > 1 {
			log.Println("Failed to allocate base image id, retrying!!!")
		}
		uid := uuid.New()
		entry, _ := s.imgTable.FindById(&uid)
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

func (s *BaseImageApiServer) baseImageList(domain string) *api.BaseImagesListResp {
	resp := &api.BaseImagesListResp{
		Items: []*api.BaseImagesListEntry{},
	}

	imageMap := map[string][]*api.BaseImagesVersionInfo{}
	list, _ := s.imgTable.GetListInDomain(domain, 0, 0)
	for _, entry := range list {
		ver := &api.BaseImagesVersionInfo{
			Version: entry.Key.Version,
			Desc:    utils.PString(entry.Desc),
			ImageId: entry.Id.String(),
			ExtRef:  entry.ExternalRef,
		}
		verList := imageMap[entry.Key.Name]
		imageMap[entry.Key.Name] = append(verList, ver)
	}

	for k, v := range imageMap {
		item := &api.BaseImagesListEntry{
			Name:     k,
			Versions: v,
		}
		resp.Items = append(resp.Items, item)
	}

	return resp
}

func (s *BaseImageApiServer) ListBaseImages(ctx context.Context, req *api.BaseImagesListReq) (*api.BaseImagesListResp, error) {
	return s.baseImageList(req.Domain), nil
}

func (s *BaseImageApiServer) AddBaseImage(ctx context.Context, req *api.BaseImageAddReq) (*api.BaseImageAddResp, error) {
	if req.Name == "" || req.Version == "" || req.ExtRef == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing mandatory parameters")
	}
	if !pattern.IsBasicName(req.Name) || !pattern.IsBasicName(req.Version) {
		return nil, status.Errorf(codes.InvalidArgument, "%q:%q invalid base image", req.Name, req.Version)
	}
	userInfo, _ := auth.GetUserInfo(ctx)
	id, err := s.allocateIdentifier()
	if err != nil {
		log.Printf("unable to allocate UUID for base image version %v, error: %s", req, err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	entry := &baseimage.BaseImageVersion{
		Key: baseimage.BaseImageVersionKey{
			Domain:  req.Domain,
			Name:    req.Name,
			Version: req.Version,
		},
		Id:          id,
		Desc:        utils.StringP(req.Desc),
		CreatedBy:   userInfo.UserName,
		ExternalRef: req.ExtRef,
	}
	err = s.imgTable.Add(entry)
	if err != nil {
		if pkgerrors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.AlreadyExists, "Entry %q already exist", entry.Key)
		}
		log.Println("Error creating base image version", entry.Key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	return &api.BaseImageAddResp{}, nil
}

func (s *BaseImageApiServer) DeleteBaseImage(ctx context.Context, req *api.BaseImageDelReq) (*api.BaseImageDelResp, error) {
	key := &baseimage.BaseImageVersionKey{
		Domain:  req.Domain,
		Name:    req.Name,
		Version: req.Version,
	}
	err := s.imgTable.Remove(key)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Entry %q not found", key)
		}
		log.Println("Error deleting base image version", key, "error", err)
		return nil, status.Errorf(codes.Internal, "Something went wrong, please try again")
	}
	return &api.BaseImageDelResp{}, nil
}

func (s *BaseImageApiServer) ListProjectBaseImages(ctx context.Context, req *api.ProjectBaseImagesListReq) (*api.BaseImagesListResp, error) {
	// project scope here is irrelavant for the function as the
	// catalog for base images will be maintained as part of
	// Domain scope itself.
	return s.baseImageList(req.Domain), nil
}
