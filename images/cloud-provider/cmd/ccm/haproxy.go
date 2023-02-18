package main

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

// proxyImage defines the loadbalancer image:tag
const proxyImage = "kindest/haproxy:v20221220-7705dd1a"

// proxyConfigPath defines the path to the config file in the image
const proxyConfigPath = "/usr/local/etc/haproxy/haproxy.cfg"

// proxyConfigData is supplied to the loadbalancer config template
type proxyConfigData struct {
	ServicePorts    []string
	HealthCheckPort int
	BackendServers  map[string]string
	IPv6            bool
}

// proxyDefaultConfigTemplate is the loadbalancer config template
const proxyDefaultConfigTemplate = `
global
  log /dev/log local0
  log /dev/log local1 notice
  daemon

resolvers docker
  nameserver dns 127.0.0.11:53

defaults
  log global
  mode tcp
  option dontlognull
  # TODO: tune these
  timeout connect 5000
  timeout client 50000
  timeout server 50000
  # allow to boot despite dns don't resolve backends
  default-server init-addr none

frontend service
{{ range $index, $port := .ServicePorts }}  bind *:{{ $port }}{{end}}
  default_backend nodes

backend nodes
  option httpchk GET /healthz
  {{- $hcport := .HealthCheckPort -}}
  {{- range $server, $address := .BackendServers }}
  server {{ $server }} {{ $address }} check port {{ $hcport }} inter 5s fall 3 rise 1
  {{- end}}
`

// proxyConfig returns a kubeadm config generated from config data, in particular
// the kubernetes version
func proxyConfig(data *proxyConfigData) (config string, err error) {
	t, err := template.New("loadbalancer-config").Parse(proxyDefaultConfigTemplate)
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
