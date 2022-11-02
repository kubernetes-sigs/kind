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
const Image = "kindest/node:v1.25.3@sha256:cd248d1438192f7814fbca8fede13cfe5b9918746dfa12583976158a834fd5c5"
