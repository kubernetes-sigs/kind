/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/exec"
)

const (
	dockerConfigEnv  = "DOCKER_CONFIG"
	dockerContextEnv = "DOCKER_CONTEXT"
	dockerHostEnv    = "DOCKER_HOST"
)

type dockerConfigFile struct {
	CurrentContext string                       `json:"currentContext"`
	Proxies        map[string]dockerProxyConfig `json:"proxies"`
}

type dockerProxyConfig struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NOProxy    string `json:"noProxy"`
}

func dockerConfigProxyEnvs(
	getEnv func(string) string,
	readFile func(string) ([]byte, error),
	inspectContextHost func(string) (string, error),
) map[string]string {
	configPath := dockerConfigPath(getEnv)
	if configPath == "" {
		return map[string]string{}
	}

	rawConfig, err := readFile(configPath)
	if err != nil {
		return map[string]string{}
	}

	cfg := dockerConfigFile{}
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return map[string]string{}
	}

	proxyCfg, ok := dockerProxySettings(cfg.Proxies, dockerHostForProxyLookup(getEnv, cfg, inspectContextHost))
	if !ok {
		return map[string]string{}
	}

	envs := map[string]string{}
	setProxyEnv(envs, common.HTTPProxy, proxyCfg.HTTPProxy)
	setProxyEnv(envs, common.HTTPSProxy, proxyCfg.HTTPSProxy)
	setProxyEnv(envs, common.NOProxy, proxyCfg.NOProxy)
	return envs
}

func dockerConfigPath(getEnv func(string) string) string {
	// Docker CLI config defaults to ~/.docker/config.json and can be overridden
	// with DOCKER_CONFIG. The same config file also stores the current context.
	// See https://docs.docker.com/reference/cli/docker/
	if dockerConfigDir := getEnv(dockerConfigEnv); dockerConfigDir != "" {
		return filepath.Join(dockerConfigDir, "config.json")
	}
	if homeDir := getEnv("HOME"); homeDir != "" {
		return filepath.Join(homeDir, ".docker", "config.json")
	}
	return ""
}

func dockerHostForProxyLookup(
	getEnv func(string) string,
	cfg dockerConfigFile,
	inspectContextHost func(string) (string, error),
) string {
	if dockerHost := getEnv(dockerHostEnv); dockerHost != "" {
		return dockerHost
	}

	contextName := getEnv(dockerContextEnv)
	if contextName == "" {
		contextName = cfg.CurrentContext
	}
	if contextName == "" {
		contextName = "default"
	}

	host, err := inspectContextHost(contextName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(host)
}

func dockerProxySettings(proxies map[string]dockerProxyConfig, dockerHost string) (dockerProxyConfig, bool) {
	if len(proxies) == 0 {
		return dockerProxyConfig{}, false
	}
	// Docker's proxies config supports a "default" entry plus per-daemon entries
	// keyed by the daemon host string, for example
	// "https://manager1.mycorp.example.com:2377".
	// See https://docs.docker.com/reference/cli/docker/
	if dockerHost != "" {
		if proxyCfg, ok := proxies[dockerHost]; ok {
			return proxyCfg, true
		}
	}
	proxyCfg, ok := proxies["default"]
	return proxyCfg, ok
}

func setProxyEnv(envs map[string]string, name, value string) {
	if value == "" {
		return
	}
	envs[name] = value
	envs[strings.ToLower(name)] = value
}

func kindProxyEnvOverridesDockerConfig(getEnv func(string) string) bool {
	for _, name := range []string{common.HTTPProxy, common.HTTPSProxy} {
		if getEnv(name) != "" || getEnv(strings.ToLower(name)) != "" {
			return true
		}
	}
	return false
}

func inspectDockerContextHost(contextName string) (string, error) {
	format := `{{ (index .Endpoints "docker").Host }}`
	lines, err := exec.OutputLines(exec.Command("docker", "context", "inspect", "--format", format, contextName))
	if err != nil || len(lines) == 0 {
		return "", err
	}
	return lines[0], nil
}
