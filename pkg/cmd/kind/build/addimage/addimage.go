/*
Copyright 2024 The Kubernetes Authors.

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

package addimage

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/apis/config/defaults"
	"sigs.k8s.io/kind/pkg/build/addimage"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Image     string
	BaseImage string
	Arch      string
	Pull      bool
}

// NewCommand returns a new cobra.Command for adding images to the node image
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.MinimumNArgs(1),
		Use:   "add-image <IMAGE> [IMAGE...]",
		Short: "Update node image with extra images preloaded",
		Long:  "Use an existing node image as a base and preload extra container images.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags, args)
		},
	}

	cmd.Flags().StringVar(
		&flags.Image, "image",
		addimage.DefaultImage,
		"name:tag of the resulting image to be built",
	)
	cmd.Flags().StringVar(
		&flags.BaseImage, "base-image",
		defaults.Image,
		"name:tag of the base image to use for the build",
	)
	cmd.Flags().StringVar(
		&flags.Arch, "arch",
		"",
		"architecture to build for, defaults to the host architecture",
	)
	return cmd
}

func runE(logger log.Logger, flags *flagpole, images []string) error {
	err := addimage.Build(
		addimage.WithImage(flags.Image),
		addimage.WithBaseImage(flags.BaseImage),
		addimage.WithAdditonalImages(images),
		addimage.WithLogger(logger),
		addimage.WithArch(flags.Arch),
	)

	return errors.Wrap(err, "error adding images to node image")
}
