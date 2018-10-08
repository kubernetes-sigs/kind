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

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/util"
)

type flags struct {
	Name      string
	Config    string
	ImageName string
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
	cmd.Flags().StringVar(&flags.Name, "name", "1", "the cluster context name")
	cmd.Flags().StringVar(&flags.Config, "config", "", "path to a kind config file")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	return cmd
}

func run(flags *flags, cmd *cobra.Command, args []string) {
	// TODO(bentheelder): make this more configurable
	// load the config
	cfg, err := encoding.Load(flags.Config)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	// validate the config
	err = cfg.Validate()
	if err != nil {
		log.Error("Invalid configuration!")
		configErrors := err.(*util.Errors)
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
	convertedCfg := cfg.ToCurrent()
	if flags.ImageName != "" {
		convertedCfg.Image = flags.ImageName
		err := convertedCfg.Validate()
		if err != nil {
			log.Errorf("Invalid flags, configuration failed validation: %v", err)
			log.Fatal("Aborting due to invalid configuration.")
		}
	}
	err = ctx.Create(convertedCfg)
	if err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}
}
