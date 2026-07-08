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

package loadbalancer

import (
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		data            *ConfigData
		configTemplate  string
		wantConfig      string
		wantErrContains string
	}{
		{
			name: "renders lds template",
			data: &ConfigData{
				ControlPlanePort: 6443,
				BackendServers: map[string]string{
					"node0": "node0:6443",
				},
				IPv6: false,
			},
			configTemplate: ProxyLDSConfigTemplate,
			wantConfig: "\n" +
				"resources:\n" +
				"- \"@type\": type.googleapis.com/envoy.config.listener.v3.Listener\n" +
				"  name: listener_apiserver\n" +
				"  address:\n" +
				"    socket_address:\n" +
				"      address: \"0.0.0.0\"\n" +
				"      port_value: 6443\n" +
				"  filter_chains:\n" +
				"  - filters:\n" +
				"    - name: envoy.filters.network.tcp_proxy\n" +
				"      typed_config:\n" +
				"        \"@type\": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy\n" +
				"        stat_prefix: ingress_tcp\n" +
				"        cluster: kube_apiservers\n",
		},
		{
			name: "renders cds template",
			data: &ConfigData{
				BackendServers: map[string]string{
					"node0": "node0:6443",
				},
				IPv6: false,
			},
			configTemplate: ProxyCDSConfigTemplate,
			wantConfig: "\n" +
				"resources:\n" +
				"- \"@type\": type.googleapis.com/envoy.config.cluster.v3.Cluster\n" +
				"  name: kube_apiservers\n" +
				"  connect_timeout: 0.25s\n" +
				"  type: STRICT_DNS\n" +
				"  lb_policy: ROUND_ROBIN\n" +
				"  dns_lookup_family: V4_PREFERRED\n" +
				"  health_checks:\n" +
				"  - timeout: 3s\n" +
				"    interval: 2s\n" +
				"    unhealthy_threshold: 2\n" +
				"    healthy_threshold: 1\n" +
				"    initial_jitter: 0s\n" +
				"    no_traffic_interval: 3s\n" +
				"    always_log_health_check_failures: true\n" +
				"    always_log_health_check_success: true\n" +
				"    event_log_path: /dev/stdout\n" +
				"    http_health_check:\n" +
				"      path: /healthz\n" +
				"    transport_socket_match_criteria:\n" +
				"      tls_mode: \"true\"\n" +
				"  transport_socket_matches:\n" +
				"  - name: \"health_check_tls\"\n" +
				"    match:\n" +
				"      tls_mode: \"true\"\n" +
				"    transport_socket:\n" +
				"      name: envoy.transport_sockets.tls\n" +
				"      typed_config:\n" +
				"        \"@type\": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext\n" +
				"        common_tls_context:\n" +
				"          validation_context:\n" +
				"            trust_chain_verification: ACCEPT_UNTRUSTED\n" +
				"  load_assignment:\n" +
				"    cluster_name: kube_apiservers\n" +
				"    endpoints:\n" +
				"    - lb_endpoints:\n" +
				"      - endpoint:\n" +
				"          address:\n" +
				"            socket_address:\n" +
				"              address: node0\n" +
				"              port_value: 6443\n",
		},
		{
			name: "renders cds template ipv6",
			data: &ConfigData{
				BackendServers: map[string]string{
					"node0": "node0:6443",
				},
				IPv6: true,
			},
			configTemplate: ProxyCDSConfigTemplate,
			wantConfig: "\n" +
				"resources:\n" +
				"- \"@type\": type.googleapis.com/envoy.config.cluster.v3.Cluster\n" +
				"  name: kube_apiservers\n" +
				"  connect_timeout: 0.25s\n" +
				"  type: STRICT_DNS\n" +
				"  lb_policy: ROUND_ROBIN\n" +
				"  dns_lookup_family: AUTO\n" +
				"  health_checks:\n" +
				"  - timeout: 3s\n" +
				"    interval: 2s\n" +
				"    unhealthy_threshold: 2\n" +
				"    healthy_threshold: 1\n" +
				"    initial_jitter: 0s\n" +
				"    no_traffic_interval: 3s\n" +
				"    always_log_health_check_failures: true\n" +
				"    always_log_health_check_success: true\n" +
				"    event_log_path: /dev/stdout\n" +
				"    http_health_check:\n" +
				"      path: /healthz\n" +
				"    transport_socket_match_criteria:\n" +
				"      tls_mode: \"true\"\n" +
				"  transport_socket_matches:\n" +
				"  - name: \"health_check_tls\"\n" +
				"    match:\n" +
				"      tls_mode: \"true\"\n" +
				"    transport_socket:\n" +
				"      name: envoy.transport_sockets.tls\n" +
				"      typed_config:\n" +
				"        \"@type\": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext\n" +
				"        common_tls_context:\n" +
				"          validation_context:\n" +
				"            trust_chain_verification: ACCEPT_UNTRUSTED\n" +
				"  load_assignment:\n" +
				"    cluster_name: kube_apiservers\n" +
				"    endpoints:\n" +
				"    - lb_endpoints:\n" +
				"      - endpoint:\n" +
				"          address:\n" +
				"            socket_address:\n" +
				"              address: node0\n" +
				"              port_value: 6443\n",
		},
		{
			name:            "returns wrapped parse error",
			data:            &ConfigData{},
			configTemplate:  `{{`,
			wantErrContains: "failed to parse config template",
		},
		{
			name: "returns wrapped execute error from cds template hostPort",
			data: &ConfigData{
				BackendServers: map[string]string{
					"node0": "node0",
				},
			},
			configTemplate:  ProxyCDSConfigTemplate,
			wantErrContains: "error executing config template",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Config(tt.data, tt.configTemplate)

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("Config() error = nil, want error containing %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("Config() error = %q, want to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Config() unexpected error: %v", err)
			}

			if got != tt.wantConfig {
				t.Fatalf("Config() output mismatch\n got:\n%q\nwant:\n%q", got, tt.wantConfig)
			}
		})
	}
}
