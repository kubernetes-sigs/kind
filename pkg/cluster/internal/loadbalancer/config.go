/*
Copyright 2019 The Kubernetes Authors.

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
	"bytes"
	"fmt"
	"net"
	"text/template"

	"sigs.k8s.io/kind/pkg/errors"
)

// ConfigData is supplied to the loadbalancer config template
type ConfigData struct {
	ControlPlanePort int
	BackendServers   map[string]string
	IPv6             bool
}

// proxyLDSConfigTemplate is the loadbalancer config template for listeners
const ProxyLDSConfigTemplate = `
resources:
- "@type": type.googleapis.com/envoy.config.listener.v3.Listener
  name: listener_apiserver
  address:
    socket_address:
      address: {{ if .IPv6 }}"::"{{ else }}"0.0.0.0"{{ end }}
      port_value: {{ .ControlPlanePort }}
      protocol: TCP
  filter_chains:
  - filters:
    - name: envoy.filters.network.tcp_proxy
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
        stat_prefix: ingress_tcp
        cluster: kube_apiservers
`

// proxyCDSConfigTemplate is the loadbalancer config template for clusters
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/health_check.proto#envoy-v3-api-msg-config-core-v3-healthcheck-httphealthcheck
const ProxyCDSConfigTemplate = `
resources:
- "@type": type.googleapis.com/envoy.config.cluster.v3.Cluster
  name: kube_apiservers
  connect_timeout: 0.25s
  type: STRICT_DNS
  lb_policy: ROUND_ROBIN
  dns_lookup_family: AUTO
  load_assignment:
      cluster_name: kube_apiservers
      endpoints:
      - lb_endpoints:
        {{- range $server, $address := .BackendServers }}
	{{- $hp := hostPort $address }}
        - endpoint:
            address:
              socket_address:
                address: {{ $hp.host }}
                port_value: {{ $hp.port }}
        {{- end }}
`

func hostPort(addr string) (map[string]string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	return map[string]string{"host": host, "port": port}, nil
}

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data *ConfigData, configTemplate string) (config string, err error) {
	funcs := template.FuncMap{
		"hostPort": hostPort,
	}
	t, err := template.New("loadbalancer-config").Funcs(funcs).Parse(configTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}
	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}

	return buff.String(), nil
}

func GenerateConfig(controlPlaneNodes []string, port int, isIPv6 bool) *ConfigData {
	backends := make(map[string]string)
	for i, ip := range controlPlaneNodes {
		backends[fmt.Sprintf("node%d", i)] = ip
	}

	return &ConfigData{
		ControlPlanePort: port,
		BackendServers:   backends,
		IPv6:             isIPv6,
	}
}
