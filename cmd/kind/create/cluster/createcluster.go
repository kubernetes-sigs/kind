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

// Package cluster implements the `create cluster` command
package cluster

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/util"
)

const (
	configFlagName            = "config"
	controlPlaneNodesFlagName = "control-plane-nodes"
	workerNodesFLagName       = "worker-nodes"
)

type flagpole struct {
	Name          string
	Config        string
	ImageName     string
	Workers       int32
	ControlPlanes int32
	Retain        bool
	Wait          time.Duration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a local Kubernetes cluster",
		Long:  "Creates a local Kubernetes cluster using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.Config, configFlagName, "", "path to a kind config file")
	cmd.Flags().Int32Var(&flags.ControlPlanes, controlPlaneNodesFlagName, 1, "number of control-plane nodes in the cluster")
	cmd.Flags().Int32Var(&flags.Workers, workerNodesFLagName, 0, "number of worker nodes in the cluster")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	cmd.Flags().BoolVar(&flags.Retain, "retain", false, "retain nodes for debugging when cluster creation fails")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "Wait for control plane node to be ready (default 0s)")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed(configFlagName) && (cmd.Flags().Changed(controlPlaneNodesFlagName) || cmd.Flags().Changed(workerNodesFLagName)) {
		return errors.Errorf("flag --%s can't be used in combination with --%s or --%s flags", configFlagName, controlPlaneNodesFlagName, workerNodesFLagName)
	}

	if flags.ControlPlanes < 0 || flags.Workers < 0 {
		return errors.Errorf("flags --%s and --%s should not be a negative number", controlPlaneNodesFlagName, workerNodesFLagName)
	}

	cfg, err := encoding.NewConfig(flags.ControlPlanes, flags.Workers)
	if err != nil {
		return errors.Wrap(err, "error creating config: %v")
	}

	// override the config with the one from file, if specified
	if flags.Config != "" {
		cfg, err = encoding.Load(flags.Config)
		if err != nil {
			return errors.Wrap(err, "error loading config: %v")
		}

		// validate the config
		err = cfg.Validate()
		if err != nil {
			log.Error("Invalid configuration!")
			configErrors := err.(util.Errors)
			for _, problem := range configErrors.Errors() {
				log.Error(problem)
			}
			return errors.New("aborting due to invalid configuration")
		}
	}

	// Check if the cluster name already exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if known {
		return errors.Errorf("a cluster with the name %q already exists", flags.Name)
	}

	// create a cluster context and create the cluster
	ctx := cluster.NewContext(flags.Name)
	if flags.ImageName != "" {
		// Apply image override to all the Nodes defined in Config
		// TODO(fabrizio pandini): this should be reconsidered when implementing
		//     https://github.com/kubernetes-sigs/kind/issues/133
		for i := range cfg.Nodes {
			cfg.Nodes[i].Image = flags.ImageName
		}

		err := cfg.Validate()
		if err != nil {
			log.Errorf("Invalid flags, configuration failed validation: %v", err)
			return errors.New("aborting due to invalid configuration")
		}
	}
	fmt.Printf("Creating cluster %q ...\n", flags.Name)
	if err = ctx.Create(cfg,
		create.Retain(flags.Retain),
		create.WaitForReady(flags.Wait),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
