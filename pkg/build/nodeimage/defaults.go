/*
Copyright 2020 The Kubernetes Authors.

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

package nodeimage

// DefaultImage is the default name:tag for the built image
const DefaultImage = "kindest/node:latest"

// DefaultBaseImage is the default base image used
// TODO: come up with a reasonable solution to digest pinning
// https://github.com/moby/moby/issues/43188
const DefaultBaseImage = "docker.io/kindest/base:v20230831-c80399bf"
