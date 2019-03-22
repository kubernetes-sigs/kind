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

package kube

import (
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const kubeletServiceURI = `https://raw.githubusercontent.com/kubernetes/release/master/debian/xenial/kubelet/lib/systemd/system/kubelet.service`

const kubeletService = `[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/

[Service]
ExecStart=/usr/bin/kubelet
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`

const tenKubeadmConfURI = `https://raw.githubusercontent.com/kubernetes/release/master/debian/xenial/kubeadm/channel/stable/etc/systemd/system/kubelet.service.d/post-1.10/10-kubeadm.conf`

const tenKubeadmConf = `# Note: This dropin only works with kubeadm and kubelet v1.11+
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
# This is a file that "kubeadm init" and "kubeadm join" generates at runtime, populating the KUBELET_KUBEADM_ARGS variable dynamically
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
# This is a file that the user can use for overrides of the kubelet args as a last resort. Preferably, the user should use
# the .NodeRegistration.KubeletExtraArgs object in the configuration files instead. KUBELET_EXTRA_ARGS should be sourced from this file.
EnvironmentFile=-/etc/default/kubelet
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
`

func getKubeletServiceBytes() []byte {
	return getRemoteOrDefaultBytes(kubeletServiceURI, kubeletService)
}

func getTenKubeadmConfBytes() []byte {
	return getRemoteOrDefaultBytes(tenKubeadmConfURI, tenKubeadmConf)
}

func getRemoteOrDefaultBytes(uri, defaultContent string) []byte {
	resp, err := http.Get(uri)
	if err == nil && resp.StatusCode != 200 {
		err = errors.Errorf("%s", resp.Status)
	}
	if err != nil {
		log.WithError(err).Debugf(
			"error reading %s; using default content", uri)
		return []byte(defaultContent)
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Debugf(
			"error reading %s; using default content", uri)
		return []byte(defaultContent)
	}
	return buf
}
