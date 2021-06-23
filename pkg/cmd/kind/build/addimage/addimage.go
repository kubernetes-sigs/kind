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
}

// NewCommand returns a new cobra.Command for adding images to the node image
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("a list of image names is required")
			}
			return nil
		},
		// TODO(bentheelder): more detailed usage
		Use:   "add-image <IMAGE> [IMAGE...]",
		Short: "Add images to a kind node image and build a custom node image",
		Long:  "Add images to a kind node image and build a custom node image",
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

func runE(logger log.Logger, flags *flagpole, args []string) error {

	if err := addimage.Build(
		addimage.WithImage(flags.Image),
		addimage.WithBaseImage(flags.BaseImage),
		addimage.WithAdditonalImages(args),
		addimage.WithLogger(logger),
		addimage.WithArch(flags.Arch),
	); err != nil {
		return errors.Wrap(err, "error adding images to node image")
	}
	return nil
}
