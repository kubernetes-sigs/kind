/*
Copyright 2024 The Kubernetes Authors.

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

package kubeadminit

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
	"sigs.k8s.io/kind/pkg/internal/version"
)

func TestGenerateBootstrapCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		kubeVersion   string
		skipKubeProxy bool
		want          []string
	}{
		{
			name:          "v1.23.0 without skipping kube-proxy",
			kubeVersion:   "v1.23.0",
			skipKubeProxy: false,
			want: []string{
				"init",
				"--config=/kind/kubeadm.conf",
				"--skip-token-print",
				"--v=6",
			},
		},
		{
			name:          "v1.23.0 with skipping kube-proxy",
			kubeVersion:   "v1.23.0",
			skipKubeProxy: true,
			want: []string{
				"init",
				"--config=/kind/kubeadm.conf",
				"--skip-token-print",
				"--v=6",
			},
		},
		{
			name:          "v1.22.0 without skipping kube-proxy",
			kubeVersion:   "v1.22.0",
			skipKubeProxy: false,
			want: []string{
				"init",
				"--config=/kind/kubeadm.conf",
				"--skip-token-print",
				"--v=6",
				"--skip-phases=preflight",
			},
		},
		{
			name:          "v1.22.0 with skipping kube-proxy",
			kubeVersion:   "v1.22.0",
			skipKubeProxy: true,
			want: []string{
				"init",
				"--config=/kind/kubeadm.conf",
				"--skip-token-print",
				"--v=6",
				"--skip-phases=preflight,addon/kube-proxy",
			},
		},
		{
			name:          "v1.25.0 without skipping kube-proxy",
			kubeVersion:   "v1.25.0",
			skipKubeProxy: false,
			want: []string{
				"init",
				"--config=/kind/kubeadm.conf",
				"--skip-token-print",
				"--v=6",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			kubeVersion, err := version.ParseGeneric(tc.kubeVersion)
			if err != nil {
				t.Fatalf("failed to parse kubernetes version %q: %v", tc.kubeVersion, err)
			}
			got := GenerateBootstrapCommand(kubeVersion, tc.skipKubeProxy)
			assert.DeepEqual(t, tc.want, got)
		})
	}
}
