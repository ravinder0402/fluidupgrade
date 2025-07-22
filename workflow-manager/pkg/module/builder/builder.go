package builder

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/coredgeio/compass/pkg/render"

	"github.com/coredgeio/workflow-manager/pkg/utils"
)

type ModuleFileInfo struct {
	Name    string
	Content string
}

type ModuleGitInfo struct {
	Url        string
	GitRef     string
	WorkingDir string
}

type ModuleBuilder struct {
	Name        string
	Namespace   string
	Files       []*ModuleFileInfo
	GitInfo     *ModuleGitInfo
	DockerFile  string
	RegInsecure bool
	Registry    string
	RegSecret   string

	// internal config parsed based on env variables
	ProxyEnabled bool
	HttpProxy    string
	HttpsProxy   string
	NoProxy      string

	// internal config to enforce resource limits
	EnforceResourceLimits bool
}

func (builder *ModuleBuilder) GetK8sObjects() ([]*unstructured.Unstructured, error) {
	builder.HttpProxy = utils.GetHttpProxyVal()
	builder.HttpsProxy = utils.GetHttpsProxyVal()
	builder.NoProxy = utils.GetNoProxyVal()
	if builder.HttpProxy != "" || builder.HttpsProxy != "" || builder.NoProxy != "" {
		builder.ProxyEnabled = true
	} else {
		builder.ProxyEnabled = false
	}

	builder.EnforceResourceLimits = utils.IsEnforceResourceLimits()

	data := render.MakeRenderData()
	b, err := json.Marshal(builder)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &data.Data)
	if err != nil {
		return nil, err
	}
	return render.RenderTemplateToK8s("module-builder", moduleBuilder, &data)
}
