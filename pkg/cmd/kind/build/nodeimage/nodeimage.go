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

package nodeimage

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/build/nodeimage"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Source    string
	BuildType string
	Image     string
	BaseImage string
	KubeRoot  string
	Arch      string
}

// NewCommand returns a new cobra.Command for building the node image
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.MaximumNArgs(1),
		// TODO(bentheelder): more detailed usage
		Use:   "node-image [kubernetes-source]",
		Short: "Build the node image",
		Long:  "Build the node image which contains Kubernetes build artifacts and other kind requirements",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Lookup("kube-root").Changed {
				if len(args) != 0 {
					return errors.New("passing an argument and deprecated --kube-root is not supported, please switch to just the argument")
				}
				logger.Warn("--kube-root is deprecated, please switch to passing this as an argument")
			}
			if cmd.Flags().Lookup("type").Changed {
				return errors.New("--type is no longer supported, please remove this flag")
			}
			return runE(logger, flags, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.BuildType,
		"type",
		"docker",
		"build type, default is docker",
	)
	cmd.Flags().StringVar(
		&flags.Image,
		"image",
		nodeimage.DefaultImage,
		"name:tag of the resulting image to be built",
	)
	cmd.Flags().StringVar(
		&flags.KubeRoot,
		"kube-root",
		"",
		"DEPRECATED: please switch to just the argument. Path to the Kubernetes source directory (if empty, the path is autodetected)",
	)
	cmd.Flags().StringVar(
		&flags.BaseImage,
		"base-image",
		nodeimage.DefaultBaseImage,
		"name:tag of the base image to use for the build",
	)
	cmd.Flags().StringVar(
		&flags.Arch,
		"arch",
		"",
		"architecture to build for, defaults to the host architecture",
	)
	return cmd
}

func runE(logger log.Logger, flags *flagpole, args []string) error {
	kubeRoot := flags.KubeRoot
	if len(args) > 0 {
		kubeRoot = args[0]
	}
	if err := nodeimage.Build(
		nodeimage.WithImage(flags.Image),
		nodeimage.WithBaseImage(flags.BaseImage),
		nodeimage.WithKuberoot(kubeRoot),
		nodeimage.WithLogger(logger),
		nodeimage.WithArch(flags.Arch),
	); err != nil {
		return errors.Wrap(err, "error building node image")
	}
	return nil
}
