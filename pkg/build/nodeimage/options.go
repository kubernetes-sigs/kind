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
	"sigs.k8s.io/kind/pkg/log"
)

// Option is a configuration option supplied to Build
type Option interface {
	apply(*buildContext) error
}

type optionAdapter func(*buildContext) error

func (c optionAdapter) apply(o *buildContext) error {
	return c(o)
}

// WithImage configures a build to tag the built image with `image`
func WithImage(image string) Option {
	return optionAdapter(func(b *buildContext) error {
		b.image = image
		return nil
	})
}

// WithBaseImage configures a build to use `image` as the base image
func WithBaseImage(image string) Option {
	return optionAdapter(func(b *buildContext) error {
		b.baseImage = image
		return nil
	})
}

// WithKuberoot sets the path to the Kubernetes source directory (if empty, the path will be autodetected)
func WithKuberoot(root string) Option {
	return optionAdapter(func(b *buildContext) error {
		b.kubeRoot = root
		return nil
	})
}

// WithLogger sets the logger
func WithLogger(logger log.Logger) Option {
	return optionAdapter(func(b *buildContext) error {
		b.logger = logger
		return nil
	})
}

// WithArch sets the architecture to build for
func WithArch(arch string) Option {
	return optionAdapter(func(b *buildContext) error {
		if arch != "" {
			b.arch = arch
		}
		return nil
	})
}
