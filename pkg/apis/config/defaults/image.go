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

// Package defaults contains cross-api-version configuration defaults
package defaults

// Image is the default for the Config.Image field, aka the default node image.
const Image = "kindest/node:v1.31.1@sha256:cd224d8da58d50907d1dd41d476587643dad2ffd9f6a4d96caf530fb3b9a5956"
