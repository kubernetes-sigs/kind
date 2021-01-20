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

import (
	"net/url"
	"runtime"

	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

// Build builds a node image using the supplied options
func Build(options ...Option) error {
	// default options
	ctx := &buildContext{
		mode:      DefaultMode,
		image:     DefaultImage,
		baseImage: DefaultBaseImage,
		logger:    log.NoopLogger{},
		// TODO: only host arch supported. changing this will be tricky
		arch: runtime.GOARCH,
	}

	// apply user options
	for _, option := range options {
		if err := option.apply(ctx); err != nil {
			return err
		}
	}

	// verify that we're using a supported arch
	if !supportedArch(ctx.arch) {
		return errors.Errorf("unsupported architecture %q", ctx.arch)
	}

	// locate sources if no kubernetes source was specified
	if ctx.kubeRoot == "" && ctx.mode != "release" {
		kubeRoot, err := kube.FindSource()
		if err != nil {
			return errors.Wrap(err, "error finding kuberoot")
		}
		ctx.kubeRoot = kubeRoot
	}
	// basic url sanitization
	if ctx.mode == "release" {
		if ctx.releaseUrl == "" {
			return errors.Errorf("A value for release-url is required for --type=release")
		}
		_, err := url.Parse(ctx.releaseUrl)
		if err != nil {
			return errors.Wrap(err, "error parsing release-url")
		}
	}

	// initialize bits
	builder, err := kube.NewNamedBuilder(ctx.logger, ctx.mode, ctx.kubeRoot, ctx.releaseUrl, ctx.arch)
	if err != nil {
		return err
	}
	ctx.builder = builder

	// do the actual build
	return ctx.Build()
}

func supportedArch(arch string) bool {
	switch arch {
	default:
		return false
	// currently we nominally support building node images for these
	case "amd64":
	case "arm64":
	case "ppc64le":
	}
	return true
}

// buildContext is used to build the kind node image, and contains
// build configuration
type buildContext struct {
	// option fields
	mode       string
	image      string
	baseImage  string
	logger     log.Logger
	releaseUrl string
	// non-option fields
	arch     string // TODO(bentheelder): this should be an option
	kubeRoot string
	builder  kube.Builder
}
