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

package kube

import (
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

// Builder represents and implementation of building Kubernetes
// building may constitute downloading a release
type Builder interface {
	// Build returns a Bits and any errors encountered while building Kubernetes.
	// Some implementations (upstream binaries) may use this step to obtain
	// an existing build instead
	Build() (Bits, error)
}

// NewNamedBuilder returns a new Builder by named implementation
// currently this includes:
// "bazel" -> NewBazelBuilder(kubeRoot)
// "docker" or "make" -> NewDockerBuilder(kubeRoot)
func NewNamedBuilder(logger log.Logger, name, kubeRoot, arch string) (Builder, error) {
	fn, err := nameToImpl(name)
	if err != nil {
		return nil, err
	}
	return fn(logger, kubeRoot, arch)
}

func nameToImpl(name string) (func(log.Logger, string, string) (Builder, error), error) {
	switch name {
	case "bazel":
		return NewBazelBuilder, nil
	// TODO: docker builder should be as-dockerized as possible, make builder
	// should use host go etc.
	case "docker":
		return NewDockerBuilder, nil
	case "make":
		return NewDockerBuilder, nil
	default:
	}
	return nil, errors.Errorf("no Bits implementation with name: %s", name)
}
