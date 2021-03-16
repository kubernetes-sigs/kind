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
const DefaultBaseImage = "kindest/base:v20210315-64ac2e7f@sha256:d93111ce508dd3134141449d0658de71b6789c01ad73aa340c98b7e640e74ed8"

// DefaultMode is the default kubernetes build mode for the built image
// see pkg/build/kube.Bits
const DefaultMode = "docker"
