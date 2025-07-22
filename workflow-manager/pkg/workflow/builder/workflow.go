package builder

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/coredgeio/compass/pkg/render"

	"github.com/coredgeio/workflow-manager/pkg/utils"
)

type WorkflowInputType struct {
	Name string
	// default value for the input
	Value string
}

type WorkflowOutputType struct {
	Name      string
	ValueFrom string
}

type WorkflowNodesType struct {
	ModuleName string
	Image      string
	Command    []string
	Args       []string
	Inputs     []*WorkflowInputType
	Outputs    []*WorkflowOutputType
}

type WorkflowStepInputSource struct {
	Source    string
	SourceVar string
}

type WorkflowStepInputType struct {
	Name   string
	Source *WorkflowStepInputSource
	Value  string
}

type WorkflowStepNode struct {
	NodeId string
	Module string
	Inputs []*WorkflowStepInputType
}

type WorkflowStepType struct {
	Nodes []*WorkflowStepNode
}

type WorkflowBuilder struct {
	Name           string
	Namespace      string
	ServiceAccount string
	UserInputs     map[string]string
	Nodes          map[string]*WorkflowNodesType
	Steps          []*WorkflowStepType

	// internal config parsed based on env variables
	ProxyEnabled bool
	HttpProxy    string
	HttpsProxy   string
	NoProxy      string

	// internal config to enforce resource limits
	EnforceResourceLimits bool
}

func (builder *WorkflowBuilder) GetK8sObjects() ([]*unstructured.Unstructured, error) {
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
	return render.RenderTemplateToK8s("workflow-builder", workflowBuilder, &data)
}

func (builder *WorkflowBuilder) GetTemplate() (string, error) {
	data := render.MakeRenderData()
	b, err := json.Marshal(builder)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(b, &data.Data)
	if err != nil {
		return "", err
	}
	return render.RenderTemplate("workflow-builder", workflowBuilder, &data)
}
