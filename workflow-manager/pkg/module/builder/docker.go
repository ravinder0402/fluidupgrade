package builder

import (
	"encoding/json"

	"github.com/coredgeio/compass/pkg/render"

	"github.com/coredgeio/workflow-manager/pkg/utils"
)

type DockerFileInfo struct {
	Name string
	Perm string
}

type DockerGitInfo struct {
	Url        string
	GitRef     string
	WorkingDir string
}

type DockerFileBuilder struct {
	BaseImage   string
	Files       []*DockerFileInfo
	GitInfo     *DockerGitInfo
	BuildScript []string
	EntryPoint  []string
	EnvVars     map[string]string

	// internal config parsed based on env variables
	HttpProxy  string
	HttpsProxy string
	NoProxy    string
}

func (builder *DockerFileBuilder) GetDockerFile() (string, error) {

	builder.HttpProxy = utils.GetHttpProxyVal()
	builder.HttpsProxy = utils.GetHttpsProxyVal()
	builder.NoProxy = utils.GetNoProxyVal()

	data := render.MakeRenderData()
	b, err := json.Marshal(builder)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(b, &data.Data)
	if err != nil {
		return "", err
	}
	return render.RenderTemplate("module-dockerfile", dockerfile, &data)
}
