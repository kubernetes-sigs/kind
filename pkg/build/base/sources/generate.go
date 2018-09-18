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

// Package sources contains the baked in sources kind needs to build.
// Primarily this includes the node-image Dockerfile, which should rarely
// change.
// These can be overridden with newer files at build-time, see ./../build
package sources

// We pull in the sources with go-bindata
//go:generate go-bindata -nometadata -mode=0666 -pkg=$GOPACKAGE -o=images_sources.go -ignore=(\.*README\.md)|(\.*BUILD\.bazel) -prefix=./../../../../ ./../../../../images/...
