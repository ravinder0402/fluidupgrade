package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type configType struct {
	MongoDB struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"mongodb"`
	MetricsDB struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"metricsdb"`
	InternalAuth struct {
		Realm  string `yaml:"realm"`
		Domain string `yaml:"domain"`
		User   string `yaml:"user"`
	} `yaml:"internalAuth"`
	Registry struct {
		Host   string `yaml:"host"`
		Port   string `yaml:"port"`
		Scheme string `yaml:"scheme"`
	} `yaml:"registry"`
	Workflow struct {
		ServiceAccount string `yaml:"serviceAccount"`
		Registry       struct {
			Name     string `yaml:"name"`
			Insecure bool   `yaml:"insecure"`
		} `yaml:"registry"`
	} `yaml:"workflow"`
}

var config configType

// GetMongodbHost returns configured mongodb host
func GetMongodbHost() string {
	return config.MongoDB.Host
}

// GetMongodbPort returns configured mongodb port
func GetMongodbPort() string {
	return config.MongoDB.Port
}

// GetMetricsdbHost returns configured metricsdb host
func GetMetricsdbHost() string {
	return config.MetricsDB.Host
}

// GetMetricsdbPort returns configured metricsdb port
func GetMetricsdbPort() string {
	return config.MetricsDB.Port
}

// GetInternalAuthRealm returns configured realm for internal auth
func GetInternalAuthRealm() string {
	return config.InternalAuth.Realm
}

// GetInternalAuthDomain returns configured domain for internal auth
func GetInternalAuthDomain() string {
	return config.InternalAuth.Domain
}

// GetInternalAuthUser returns configured user for internal auth
func GetInternalAuthUser() string {
	return config.InternalAuth.User
}

// GetRegistryHost returns available registry host
func GetRegistryHost() string {
	if config.Registry.Host == "" {
		return "container-registry"
	}
	return config.Registry.Host
}

// GetRegistryPort returns available registry port
func GetRegistryPort() string {
	if config.Registry.Port == "" {
		return "8080"
	}
	return config.Registry.Port
}

// GetRegistryScheme returns available registry Scheme
func GetRegistryScheme() string {
	if config.Registry.Scheme == "" {
		return "http"
	}
	return config.Registry.Scheme
}

func GetWorkflowServiceAccount() string {
	return config.Workflow.ServiceAccount
}

func GetWorkflowRegistryName() string {
	return config.Workflow.Registry.Name
}

func IsWorkflowRegistryInsecure() bool {
	return config.Workflow.Registry.Insecure
}

// validateConfigPath just makes sure, that the path provided is a file,
// that can be read
func validateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// ParseConfig returns a new decoded Config struct
func ParseConfig(configPath string) error {
	// validate config path before decoding
	if err := validateConfigPath(configPath); err != nil {
		return err
	}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return err
	}

	return nil
}
