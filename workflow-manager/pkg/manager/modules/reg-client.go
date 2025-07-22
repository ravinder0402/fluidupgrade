package modules

import (
	"io"
	"net/http"

	"github.com/coredgeio/compass/pkg/auth"
	"github.com/coredgeio/gosdkclient"
	sdkerrors "github.com/coredgeio/gosdkclient/errors"
	"github.com/coredgeio/orbiter-registry/api/registry"
)

type client struct {
	gosdkclient.SdkClient
	config RegClientConfig
}

func (c *client) PerformReq(req *http.Request) (*http.Response, error) {
	userinfo := &auth.UserAuthInformation{
		UserName:   c.config.User,
		RealmName:  c.config.Realm,
		DomainName: c.config.Domain,
		RealmAccess: struct {
			Roles []string `json:"roles,omitempty"`
		}{
			Roles: []string{
				auth.SuperAdminRoleName,
				auth.DomainAdminRoleName,
			},
		},
	}
	req.URL.Host = c.config.Host
	req.URL.Scheme = c.config.Scheme
	req.Host = c.config.Host
	err := auth.HttpRequestSetUserInformation(req, userinfo)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		// we should be acknowledging only 2xx as succesful response
		// everything else is considered as error or failure from the
		// server side
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			// handle this as error situation
			// since we won't send out the response body it is
			// our responsibility to close it over here
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			err = sdkerrors.ParseError(resp.StatusCode, bodyBytes)
			return nil, err
		}
	}

	return resp, err
}

type RegClientConfig struct {
	Realm  string
	Domain string
	User   string
	Scheme string
	Host   string
}

func (config *RegClientConfig) getRegistryClient() registry.RegistryApiSdkClient {
	c := &client{
		config: *config,
	}

	return registry.NewRegistryApiSdkClient(c)
}
