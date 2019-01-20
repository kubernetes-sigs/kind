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

package haproxy

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

// ConfigData is supplied to the haproxy config template
type ConfigData struct {
	ControlPlanePort int
	BackendServers   map[string]string
}

// DefaultConfigTemplate is the haproxy config template
const DefaultConfigTemplate = `
global
    log         127.0.0.1 local2
    daemon

defaults
    mode                    http
    log                     global
    option                  httplog
    option                  dontlognull
    option http-server-close
    option forwardfor       except 127.0.0.0/8
    option                  redispatch
    retries                 3
    timeout http-request    10s
    timeout queue           1m
    timeout connect         10s
    timeout client          1m
    timeout server          1m
    timeout http-keep-alive 10s
    timeout check           10s
    maxconn                 3000

frontend controlPlane
    bind *:{{ .ControlPlanePort }}
    option tcplog
    mode tcp
    default_backend kube-apiservers

backend kube-apiservers
    mode tcp
    balance roundrobin
    option ssl-hello-chk
{{range $server, $address := .BackendServers}}
    server {{ $server }} {{ $address }} check
{{- end}}
`

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data *ConfigData) (config string, err error) {
	t, err := template.New("haproxy-config").Parse(DefaultConfigTemplate)
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
