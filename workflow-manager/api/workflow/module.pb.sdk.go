// Code generated by protoc-gen-coredge-sdk. DO NOT EDIT.
// source: module.proto

/*
Package workflow is auto generated SDK module

It provides auto generated functions to perform operations
using APIs defined as part of protobuf
*/
package workflow

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/coredgeio/gosdkclient"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type ModuleApiSdkClient interface {
	ListModules(req *ModulesListReq) (*ModulesListResp, error)
	CreateModule(req *ModuleCreateReq) (*ModuleCreateResp, error)
	GetModule(req *ModuleEntryKey) (*ModuleGetResp, error)
	UpdateModule(req *ModuleCreateReq) (*ModuleCreateResp, error)
	DeleteModule(req *ModuleEntryKey) (*ModuleDeleteResp, error)
	RebuildModule(req *ModuleEntryKey) (*ModuleRebuildResp, error)
	CreateCatalogRequest(req *ModuleCatalogCreateReq) (*ModuleCatalogCreateResp, error)
	DeleteCatalogRequest(req *ModuleCatalogDeleteReq) (*ModuleCatalogDeleteResp, error)
	AddModuleComment(req *ModuleCommentAddReq) (*ModuleCommentAddResp, error)
	ListModuleCatalog(req *ModuleCatalogListReq) (*ModuleCatalogListResp, error)
	ListModuleCatalogVer(req *ModuleCatalogVerListReq) (*ModuleCatalogVerListResp, error)
	ListModuleCatalogRequest(req *ModuleCatalogRequestListReq) (*ModuleCatalogRequestListResp, error)
}

type implModuleApiClient struct {
	client     gosdkclient.SdkClient
	pathPrefix string
}

func NewModuleApiSdkClient(client gosdkclient.SdkClient) ModuleApiSdkClient {
	return &implModuleApiClient{
		client: client,
	}
}

func (c *implModuleApiClient) ListModules(req *ModulesListReq) (*ModulesListResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/modules"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("GET", subUrl, nil)
	q := url.Values{}
	q.Add("offset", fmt.Sprintf("%v", req.Offset))
	q.Add("limit", fmt.Sprintf("%v", req.Limit))
	q.Add("search", fmt.Sprintf("%v", req.Search))
	r.URL.RawQuery = q.Encode()
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModulesListResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) CreateModule(req *ModuleCreateReq) (*ModuleCreateResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	marshaller := &runtime.JSONPb{}
	jsonData, _ := marshaller.Marshal(req)
	r, _ := http.NewRequest("POST", subUrl, bytes.NewBuffer(jsonData))
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCreateResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) GetModule(req *ModuleEntryKey) (*ModuleGetResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("GET", subUrl, nil)
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleGetResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) UpdateModule(req *ModuleCreateReq) (*ModuleCreateResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	jsonData, _ := marshaller.Marshal(req)
	r, _ := http.NewRequest("PUT", subUrl, bytes.NewBuffer(jsonData))
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCreateResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) DeleteModule(req *ModuleEntryKey) (*ModuleDeleteResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("DELETE", subUrl, nil)
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleDeleteResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) RebuildModule(req *ModuleEntryKey) (*ModuleRebuildResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}/rebuild"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("POST", subUrl, nil)
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleRebuildResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) CreateCatalogRequest(req *ModuleCatalogCreateReq) (*ModuleCatalogCreateResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}/catalog"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	jsonData, _ := marshaller.Marshal(req)
	r, _ := http.NewRequest("POST", subUrl, bytes.NewBuffer(jsonData))
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCatalogCreateResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) DeleteCatalogRequest(req *ModuleCatalogDeleteReq) (*ModuleCatalogDeleteResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}/catalog"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("DELETE", subUrl, nil)
	q := url.Values{}
	q.Add("msg", fmt.Sprintf("%v", req.Msg))
	r.URL.RawQuery = q.Encode()
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCatalogDeleteResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) AddModuleComment(req *ModuleCommentAddReq) (*ModuleCommentAddResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module/{name}/comment"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	jsonData, _ := marshaller.Marshal(req)
	r, _ := http.NewRequest("POST", subUrl, bytes.NewBuffer(jsonData))
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCommentAddResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) ListModuleCatalog(req *ModuleCatalogListReq) (*ModuleCatalogListResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module-catalog"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("GET", subUrl, nil)
	q := url.Values{}
	q.Add("offset", fmt.Sprintf("%v", req.Offset))
	q.Add("limit", fmt.Sprintf("%v", req.Limit))
	r.URL.RawQuery = q.Encode()
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCatalogListResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) ListModuleCatalogVer(req *ModuleCatalogVerListReq) (*ModuleCatalogVerListResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/project/{project}/module-catalog/{name}/versions"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	subUrl = strings.Replace(subUrl, "{"+"project"+"}", fmt.Sprintf("%v", req.Project), -1)
	subUrl = strings.Replace(subUrl, "{"+"name"+"}", fmt.Sprintf("%v", req.Name), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("GET", subUrl, nil)
	q := url.Values{}
	q.Add("offset", fmt.Sprintf("%v", req.Offset))
	q.Add("limit", fmt.Sprintf("%v", req.Limit))
	r.URL.RawQuery = q.Encode()
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCatalogVerListResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *implModuleApiClient) ListModuleCatalogRequest(req *ModuleCatalogRequestListReq) (*ModuleCatalogRequestListResp, error) {
	// TODO(prabhjot) we are ignoring the error here for the time being
	subUrl := "/api/workflow/v1/domain/{domain}/catalog-requests"
	subUrl = strings.Replace(subUrl, "{"+"domain"+"}", fmt.Sprintf("%v", req.Domain), -1)
	marshaller := &runtime.JSONPb{}
	r, _ := http.NewRequest("GET", subUrl, nil)
	q := url.Values{}
	q.Add("offset", fmt.Sprintf("%v", req.Offset))
	q.Add("limit", fmt.Sprintf("%v", req.Limit))
	q.Add("status", fmt.Sprintf("%v", req.Status))
	r.URL.RawQuery = q.Encode()
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.PerformReq(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj := &ModuleCatalogRequestListResp{}
	err = marshaller.Unmarshal(bodyBytes, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
