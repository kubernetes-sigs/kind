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

// APIVersion is the kubernetes-style API apiVersion for this Config package
const APIVersion = "kind.sigs.k8s.io/v1alpha1"

// ConfigKind is the kubernetes-style API kind identifier for Config
const ConfigKind = "Config"

// Kind returns the `kind:` for Config
func (c *Config) Kind() string {
	return ConfigKind
}

// APIVersion returns the `apiVersion:` for Config
func (c *Config) APIVersion() string {
	return SchemeGroupVersion.String()
}
