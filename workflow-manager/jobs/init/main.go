package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/coredgeio/compass/pkg/auth"
	"github.com/coredgeio/gosdkclient"
	"github.com/coredgeio/gosdkclient/errors"
	"github.com/coredgeio/orbiter-auth/api/access"
	"github.com/coredgeio/orbiter-registry/api/registry"

	"github.com/coredgeio/workflow-manager/jobs/pkg/k8s/secret"
)

var (
	authHost  string
	regHost   string
	regModule string
	urlScheme string
	userName  string
	userRealm string
	secureReg bool
)

type dockerAuth struct {
	Auth []byte `json:"auth,omitempty"`
}

type dockerConfig struct {
	Auths map[string]dockerAuth `json:"auths,omitempty"`
}

type client struct {
	gosdkclient.SdkClient
	host string
}

func (c *client) PerformReq(req *http.Request) (*http.Response, error) {
	userinfo := &auth.UserAuthInformation{
		UserName:   userName,
		RealmName:  userRealm,
		DomainName: "default",
		RealmAccess: struct {
			Roles []string `json:"roles,omitempty"`
		}{
			Roles: []string{
				auth.SuperAdminRoleName,
				auth.DomainAdminRoleName,
			},
		},
	}
	req.URL.Host = c.host
	req.URL.Scheme = urlScheme
	req.Host = c.host
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
			err = errors.ParseError(resp.StatusCode, bodyBytes)
			return nil, err
		}
	}

	return resp, err
}

// parseFlags will create and parse the CLI flags
func parseFlags() error {
	flag.StringVar(&authHost, "host", "", "Host endpoint for auth module")
	flag.StringVar(&regHost, "reg-host", "", "registry endpoint for pushing container images")
	flag.StringVar(&regModule, "reg-internal-host", "container-registry:8080", "Internal Host endpoint for registry module")
	flag.StringVar(&urlScheme, "scheme", "http", "Host url scheme for auth module: default is http")
	flag.StringVar(&userName, "user", "", "username to be used")
	flag.StringVar(&userRealm, "realm", "", "realm / tenant for which the domain will be created")
	flag.BoolVar(&secureReg, "secure-reg", false, "if the registry enpoint is secure")

	// Actually parse the flags
	flag.Parse()

	if authHost == "" || userName == "" || userRealm == "" {
		return errors.Wrap("invalid arguments")
	}

	return nil
}

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatalln(err)
	}

	c := &client{
		host: authHost,
	}
	domClient := access.NewDomainApiSdkClient(c)
	req := &access.DomainListReq{}
	resp, err := domClient.ListDomains(req)
	if err != nil {
		log.Fatalln("failed to get list of available domains", err)
	}
	if len(resp.Items) == 0 {
		// create a new domain
		req := &access.DomainCreateReq{
			Name: "default",
		}
		_, err := domClient.CreateDomain(req)
		if err != nil {
			log.Fatalln("failed to create default domain", err)
		}
	} else {
		log.Println("got resp", resp)
		log.Println("system already has configured domains, skipping domain creation")
	}

	// TODO(prabhjot) need to remove the explicit deletion after next upgrade
	err = secret.DeleteRegistrySecret()
	if err != nil {
		log.Println("failed to remove old registry secret", err)
	}

	// check if registry secret is already available
	if !secret.IsRegistrySecretCreated() {
		c.host = regModule
		regClient := registry.NewRegistryApiSdkClient(c)

		listReq := &registry.RegistryListReq{
			Domain: "default",
		}
		listResp, err := regClient.ListRegistries(listReq)
		if err != nil {
			log.Println("failed to fetch available catalog registry", err)
		}
		if listResp == nil || len(listResp.Items) == 0 {
			// create a new registry
			req := &registry.RegistryCreateReq{
				Domain:            "default",
				Name:              "catalog",
				Plan:              registry.PlanType_Professional,
				DefaultVisibility: registry.RegistryVisibilityScope_Public,
			}
			_, err := regClient.CreateRegistry(req)
			if err != nil {
				log.Fatalln("failed to create catalog registry", err)
			}
		}

		c.host = authHost
		accClient := access.NewMyAccountApiSdkClient(c)
		delReq := &access.RegistryTokenDeleteReq{
			Name: "internal-token",
		}
		_, _ = accClient.DeleteRegistryToken(delReq)

		tokenReq := &access.RegistryTokenCreateReq{
			Name:  "internal-token",
			Scope: access.RegistryTokenScope_ReadWrite,
		}
		tokenResp, err := accClient.CreateRegistryToken(tokenReq)
		if err != nil {
			log.Fatalln("failed to create registry token", err)
		}

		auth := &dockerConfig{
			Auths: map[string]dockerAuth{
				regHost: dockerAuth{
					Auth: []byte(userName + ":" + tokenResp.Secret),
				},
			},
		}
		config, _ := json.Marshal(auth)
		err = secret.EnsureRegistrySecret(config)
		if err != nil {
			log.Println("failed to create registry secret", err)
		}
	}

	log.Println("init successfully completed!")
}
