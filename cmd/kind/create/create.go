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

// Package create implements the `create` command
package create

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/pkg/cluster"
)

type flags struct {
	Name   string
	Config string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flags{}
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "create",
		Short: "Creates a cluster",
		Long:  "Creates a Kubernetes cluster",
		Run: func(cmd *cobra.Command, args []string) {
			run(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "1", "the cluster name")
	cmd.Flags().StringVar(&flags.Config, "config", "", "path to create config file")
	return cmd
}

func run(flags *flags, cmd *cobra.Command, args []string) {
	// TODO(bentheelder): make this more configurable
	// load the config
	config, err := cluster.LoadCreateConfig(flags.Config)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	// validate the config
	err = config.Validate()
	if err != nil {
		log.Error("Invalid configuration!")
		configErrors := err.(cluster.ConfigErrors)
		for _, problem := range configErrors.Errors() {
			log.Error(problem)
		}
		log.Fatal("Aborting due to invalid configuration.")
	}
	// create a cluster context and create the cluster
	ctx, err := cluster.NewContext(flags.Name)
	if err != nil {
		log.Fatalf("Failed to create cluster context! %v", err)
	}
	err = ctx.Create(config)
	if err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}
}
