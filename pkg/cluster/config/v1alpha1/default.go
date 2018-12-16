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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DefaultImage is the default for the Config.Image field, aka the default node
// image.
const DefaultImage = "kindest/node:v1.12.3@sha256:f0ecb1066697d9417365ca58410132e512ce2010763470bb28c1e8f7fef55464"

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Config sets uninitialized fields to their default value.
func SetDefaults_Config(obj *Config) {
	if obj.Image == "" {
		obj.Image = DefaultImage
	}
}
