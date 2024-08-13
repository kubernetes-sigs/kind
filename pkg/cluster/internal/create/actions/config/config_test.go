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

// Package config implements the kubeadm config action
package config

import (
	"log"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	internalencoding "sigs.k8s.io/kind/pkg/internal/apis/config/encoding"
	"sigs.k8s.io/kind/pkg/internal/patch"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	cases := []struct {
		Name      string
		Input     string
		SubString string
	}{
		{
			Name: "Check if extraEnvs takes effect",
			Input: `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 6443
        hostPort: 6443
        protocol: TCP
  - role: worker
kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      certSANs:
        - localhost
      extraEnvs:
        - name: OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
          value: http://10.230.205.190:5080/api/default/traces
`, SubString: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		},
	}

	for _, tc := range cases {
		tc := tc // capture tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			cfg, err := internalencoding.Parse([]byte(tc.Input))
			data := kubeadm.ConfigData{
				ClusterName:       "test",
				KubernetesVersion: "v1.31.0",
			}
			cf, err := kubeadm.Config(data)
			if err != nil {
				log.Fatalln(err)
			}
			clusterPatches, clusterJSONPatches := allPatchesFromConfig(cfg)
			// apply cluster-level patches first
			patchedConfig, err := patch.KubeYAML(cf, clusterPatches, clusterJSONPatches)
			if err != nil {
				log.Fatalln(err)
			}
			// if needed, apply current node's patches
			if len(cfg.KubeadmConfigPatches) > 0 || len(cfg.KubeadmConfigPatchesJSON6902) > 0 {
				patchedConfig, _ = patch.KubeYAML(patchedConfig, cfg.KubeadmConfigPatches, cfg.KubeadmConfigPatchesJSON6902)
			}
			if !strings.Contains(patchedConfig, tc.SubString) {
				log.Fatalln(tc.Name, "invalid")
			}
		})
	}

}g
