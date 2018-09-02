/*
Copyright 2018 The Kubernetes Authors.

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

package config

import (
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	cases := []struct {
		TestName string
		Config   *Config
	}{
		{
			TestName: "Canonical config",
			Config:   New(),
		},
		{
			TestName: "Config with NodeLifecyle hooks",
			Config: func() *Config {
				cfg := New()
				cfg.NodeLifecycle = &NodeLifecycle{
					PreBoot: []LifecycleHook{
						{
							Command: "ps",
							Args:    []string{"aux"},
						},
					},
					PreKubeadm: []LifecycleHook{
						{
							Name:    "docker ps",
							Command: "docker",
							Args:    []string{"ps"},
						},
					},
					PostKubeadm: []LifecycleHook{
						{
							Name:    "docker ps again",
							Command: "docker",
							Args:    []string{"ps", "-a"},
						},
					},
				}
				return cfg
			}(),
		},
	}
	for _, tc := range cases {
		original := tc.Config
		deepCopy := tc.Config.DeepCopy()
		if !reflect.DeepEqual(original, deepCopy) {
			t.Errorf(
				"case: '%s' deep copy did not equal original: %+v != %+v",
				tc.TestName, original, deepCopy,
			)
		}
	}
}
