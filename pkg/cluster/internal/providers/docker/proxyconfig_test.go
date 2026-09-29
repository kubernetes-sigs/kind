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
	"path/filepath"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestDockerConfigProxyEnvs(t *testing.T) {
	t.Parallel()

	const configJSON = `{
		"proxies": {
			"default": {
				"httpProxy": "http://proxy.default:3128",
				"httpsProxy": "https://proxy.default:3129",
				"noProxy": "default.internal"
			},
			"tcp://remote.example.com:2376": {
				"httpProxy": "http://proxy.remote:4128",
				"httpsProxy": "https://proxy.remote:4129",
				"noProxy": "remote.internal"
			}
		}
	}`

	cases := []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{
			name: "uses default docker config",
			env: map[string]string{
				"DOCKER_CONFIG": "/tmp/docker-config",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.default:3128",
				"http_proxy":      "http://proxy.default:3128",
				common.HTTPSProxy: "https://proxy.default:3129",
				"https_proxy":     "https://proxy.default:3129",
				common.NOProxy:    "default.internal",
				"no_proxy":        "default.internal",
			},
		},
		{
			name: "uses host specific docker config when DOCKER_HOST matches",
			env: map[string]string{
				"DOCKER_CONFIG": "/tmp/docker-config",
				"DOCKER_HOST":   "tcp://remote.example.com:2376",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.remote:4128",
				"http_proxy":      "http://proxy.remote:4128",
				common.HTTPSProxy: "https://proxy.remote:4129",
				"https_proxy":     "https://proxy.remote:4129",
				common.NOProxy:    "remote.internal",
				"no_proxy":        "remote.internal",
			},
		},
		{
			name: "uses host specific docker config when DOCKER_CONTEXT matches a remote host",
			env: map[string]string{
				"DOCKER_CONFIG":  "/tmp/docker-config",
				"DOCKER_CONTEXT": "remote-context",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.remote:4128",
				"http_proxy":      "http://proxy.remote:4128",
				common.HTTPSProxy: "https://proxy.remote:4129",
				"https_proxy":     "https://proxy.remote:4129",
				common.NOProxy:    "remote.internal",
				"no_proxy":        "remote.internal",
			},
		},
		{
			name: "uses host specific docker config when currentContext matches a remote host",
			env: map[string]string{
				"DOCKER_CONFIG": "/tmp/docker-config",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.remote:4128",
				"http_proxy":      "http://proxy.remote:4128",
				common.HTTPSProxy: "https://proxy.remote:4129",
				"https_proxy":     "https://proxy.remote:4129",
				common.NOProxy:    "remote.internal",
				"no_proxy":        "remote.internal",
			},
		},
		{
			name: "falls back to HOME docker config path",
			env: map[string]string{
				"HOME": "/home/tester",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.default:3128",
				"http_proxy":      "http://proxy.default:3128",
				common.HTTPSProxy: "https://proxy.default:3129",
				"https_proxy":     "https://proxy.default:3129",
				common.NOProxy:    "default.internal",
				"no_proxy":        "default.internal",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfgJSON := configJSON
			if tc.name == "uses host specific docker config when currentContext matches a remote host" {
				cfgJSON = `{
		"currentContext": "remote-context",
		"proxies": {
			"default": {
				"httpProxy": "http://proxy.default:3128",
				"httpsProxy": "https://proxy.default:3129",
				"noProxy": "default.internal"
			},
			"tcp://remote.example.com:2376": {
				"httpProxy": "http://proxy.remote:4128",
				"httpsProxy": "https://proxy.remote:4129",
				"noProxy": "remote.internal"
			}
		}
	}`
			}

			result := dockerConfigProxyEnvs(func(key string) string {
				return tc.env[key]
			}, func(name string) ([]byte, error) {
				switch name {
				case filepath.Join("/tmp/docker-config", "config.json"), filepath.Join("/home/tester", ".docker", "config.json"):
					return []byte(cfgJSON), nil
				default:
					t.Fatalf("unexpected config path %q", name)
					return nil, nil
				}
			}, func(contextName string) (string, error) {
				switch contextName {
				case "default":
					return "unix:///var/run/docker.sock", nil
				case "remote-context":
					return "tcp://remote.example.com:2376", nil
				default:
					t.Fatalf("unexpected context %q", contextName)
					return "", nil
				}
			})

			assert.DeepEqual(t, tc.want, result)
		})
	}
}

func TestDockerConfigProxyEnvsIgnoresInvalidConfig(t *testing.T) {
	t.Parallel()

	result := dockerConfigProxyEnvs(func(key string) string {
		if key == "DOCKER_CONFIG" {
			return "/tmp/docker-config"
		}
		return ""
	}, func(name string) ([]byte, error) {
		return []byte("{not-json"), nil
	}, func(string) (string, error) {
		t.Fatal("context inspection should not be called for invalid config")
		return "", nil
	})

	assert.DeepEqual(t, map[string]string{}, result)
}

func TestKindProxyEnvOverridesDockerConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "http proxy set",
			env: map[string]string{
				common.HTTPProxy: "http://proxy.example.com:3128",
			},
			want: true,
		},
		{
			name: "https proxy set in lower case",
			env: map[string]string{
				"https_proxy": "https://proxy.example.com:3129",
			},
			want: true,
		},
		{
			name: "only no proxy set",
			env: map[string]string{
				common.NOProxy: "example.internal",
			},
			want: false,
		},
		{
			name: "no proxy env set",
			env:  map[string]string{},
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := kindProxyEnvOverridesDockerConfig(func(key string) string {
				return tc.env[key]
			})
			assert.BoolEqual(t, tc.want, result)
		})
	}
}

func TestMergeProxyEnvWithDockerConfig(t *testing.T) {
	t.Parallel()

	cluster := &config.Cluster{}
	cluster.Networking.ServiceSubnet = "10.96.0.0/16"
	cluster.Networking.PodSubnet = "10.244.0.0/16"

	const configJSON = `{
		"proxies": {
			"default": {
				"httpProxy": "http://proxy.default:3128",
				"httpsProxy": "https://proxy.default:3129",
				"noProxy": "default.internal"
			}
		}
	}`

	cases := []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{
			name: "uses docker config when shell proxy env is unset",
			env: map[string]string{
				"DOCKER_CONFIG": "/tmp/docker-config",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.default:3128",
				"http_proxy":      "http://proxy.default:3128",
				common.HTTPSProxy: "https://proxy.default:3129",
				"https_proxy":     "https://proxy.default:3129",
				common.NOProxy:    "default.internal,10.96.0.0/16,10.244.0.0/16",
				"no_proxy":        "default.internal,10.96.0.0/16,10.244.0.0/16",
			},
		},
		{
			name: "shell no proxy is preserved while docker fills http vars",
			env: map[string]string{
				"DOCKER_CONFIG": "/tmp/docker-config",
				common.NOProxy:  "shell.internal",
			},
			want: map[string]string{
				common.HTTPProxy:  "http://proxy.default:3128",
				"http_proxy":      "http://proxy.default:3128",
				common.HTTPSProxy: "https://proxy.default:3129",
				"https_proxy":     "https://proxy.default:3129",
				common.NOProxy:    "shell.internal,10.96.0.0/16,10.244.0.0/16",
				"no_proxy":        "shell.internal,10.96.0.0/16,10.244.0.0/16",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			baseEnv := map[string]string{}
			for _, name := range []string{common.HTTPProxy, common.HTTPSProxy, common.NOProxy, "http_proxy", "https_proxy", "no_proxy"} {
				if value := tc.env[name]; value != "" {
					baseEnv[name] = value
				}
			}
			if len(baseEnv) > 0 {
				noProxy := baseEnv[common.NOProxy]
				if noProxy == "" {
					noProxy = baseEnv["no_proxy"]
				}
				if noProxy != "" {
					noProxy += ","
				}
				noProxy += cluster.Networking.ServiceSubnet + "," + cluster.Networking.PodSubnet
				baseEnv[common.NOProxy] = noProxy
				baseEnv["no_proxy"] = noProxy
			}

			result := mergeProxyEnvWithDockerConfig(cluster, baseEnv, dockerConfigProxyEnvs(func(key string) string {
				return tc.env[key]
			}, func(name string) ([]byte, error) {
				if name != filepath.Join("/tmp/docker-config", "config.json") {
					t.Fatalf("unexpected config path %q", name)
				}
				return []byte(configJSON), nil
			}, func(string) (string, error) {
				return "", nil
			}))
			assert.DeepEqual(t, tc.want, result)
		})
	}
}
