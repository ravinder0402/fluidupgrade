package utils

import (
	"os"
	"strconv"
)

func IsEnforceResourceLimits() bool {
	val, ok := os.LookupEnv("ENFORCE_RESOURCE_LIMITS")
	if ok {
		ret, _ := strconv.ParseBool(val)
		return ret
	}
	return false
}

func getEnvAny(names ...string) string {
	for _, n := range names {
		if val := os.Getenv(n); val != "" {
			return val
		}
	}
	return ""
}

func GetHttpProxyVal() string {
	return getEnvAny("HTTP_PROXY", "http_proxy")
}

func GetHttpsProxyVal() string {
	return getEnvAny("HTTPS_PROXY", "https_proxy")
}

func GetNoProxyVal() string {
	return getEnvAny("NO_PROXY", "no_proxy")
}
