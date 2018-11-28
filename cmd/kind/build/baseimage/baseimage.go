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

package baseimage

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/build/base"
)

type flagpole struct {
	Source string
	Image  string
}

// NewCommand returns a new cobra.Command for building the base image
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "base-image",
		Short: "build the base node image",
		Long:  `build the base node image for running nested containers, systemd, and kubernetes components.`,
		Run: func(cmd *cobra.Command, args []string) {
			run(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.Source, "source",
		"",
		"path to the base image sources",
	)
	cmd.Flags().StringVar(
		&flags.Image, "image",
		base.DefaultImage,
		"name:tag of the resulting image to be built",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// TODO(bentheelder): make this more configurable
	ctx := base.NewBuildContext(
		base.WithImage(flags.Image),
		base.WithSourceDir(flags.Source),
	)
	err := ctx.Build()
	if err != nil {
		log.Fatalf("Build failed! %v", err)
	}
}
