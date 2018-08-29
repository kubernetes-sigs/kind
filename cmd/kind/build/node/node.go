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

package node

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/pkg/build"
)

type flags struct {
	Source    string
	BuildType string
}

// NewCommand returns a new cobra.Command for building the node image
func NewCommand() *cobra.Command {
	flags := &flags{}
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "node",
		Short: "build the node image",
		Long:  "build the node image",
		Run: func(cmd *cobra.Command, args []string) {
			run(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.BuildType, "type", "docker", "build type, one of [bazel, docker, apt]")
	return cmd
}

func run(flags *flags, cmd *cobra.Command, args []string) {
	// TODO(bentheelder): make this more configurable
	ctx, err := build.NewNodeImageBuildContext(flags.BuildType)
	if err != nil {
		log.Fatalf("Error creating build context: %v", err)
	}
	err = ctx.Build()
	if err != nil {
		log.Fatalf("Error building node image: %v", err)
	}
}
