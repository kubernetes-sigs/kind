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

package patch

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestTOML(t *testing.T) {
	t.Parallel()
	type testCase struct {
		Name            string
		ToPatch         string
		Patches         []string
		PatchesJSON6902 []string
		ExpectError     bool
		ExpectOutput    string
	}
	cases := []testCase{
		{
			Name:         "invalid TOML",
			ToPatch:      `🗿`,
			ExpectError:  true,
			ExpectOutput: "",
		},
		{
			Name: "no patches",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			ExpectError: false,
			ExpectOutput: `disabled_plugins = ["restart"]

[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
  [plugins.linux]
    shim_debug = true
`,
		},
		{
			Name: "invalid patch TOML",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			Patches:     []string{"🏰"},
			ExpectError: true,
		},
		{
			Name: "invalid 6902 patch JSON",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			PatchesJSON6902: []string{"🏰"},
			ExpectError:     true,
		},
		{
			Name: "trivial patch",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			Patches:     []string{`disabled_plugins=[]`},
			ExpectError: false,
			ExpectOutput: `disabled_plugins = []

[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
  [plugins.linux]
    shim_debug = true
`,
		},
		{
			Name: "trivial 6902 patch",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			PatchesJSON6902: []string{`[{"op": "remove", "path": "/disabled_plugins"}]`},
			ExpectError:     false,
			ExpectOutput: `[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
  [plugins.linux]
    shim_debug = true
`,
		},
		{
			Name: "trivial patch and trivial 6902 patch",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			Patches:         []string{`disabled_plugins=["foo"]`},
			PatchesJSON6902: []string{`[{"op": "remove", "path": "/disabled_plugins"}]`},
			ExpectError:     false,
			ExpectOutput: `[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
  [plugins.linux]
    shim_debug = true
`,
		},
		{
			Name: "invalid path 6902 patch",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			PatchesJSON6902: []string{`[{"op": "remove", "path": "/fooooooo"}]`},
			ExpectError:     true,
			ExpectOutput: `[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
  [plugins.linux]
    shim_debug = true
`,
		},
		{
			Name: "patch registry",
			ToPatch: `disabled_plugins = ["restart"]
[plugins.linux]
  shim_debug = true
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"`,
			Patches: []string{`[plugins.cri.registry.mirrors]
  [plugins.cri.registry.mirrors."registry:5000"]
    endpoint = ["http://registry:5000"]`},
			ExpectError: false,
			ExpectOutput: `disabled_plugins = ["restart"]

[plugins]
  [plugins.cri]
    [plugins.cri.containerd]
      [plugins.cri.containerd.runtimes]
        [plugins.cri.containerd.runtimes.runsc]
          runtime_type = "io.containerd.runsc.v1"
    [plugins.cri.registry]
      [plugins.cri.registry.mirrors]
        [plugins.cri.registry.mirrors."registry:5000"]
          endpoint = ["http://registry:5000"]
  [plugins.linux]
    shim_debug = true
`,
		},
	}
	for _, tc := range cases {
		tc := tc // capture test case
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			out, err := TOML(tc.ToPatch, tc.Patches, tc.PatchesJSON6902)
			assert.ExpectError(t, tc.ExpectError, err)
			if err == nil {
				assert.StringEqual(t, tc.ExpectOutput, out)
			}
		})
	}
}
