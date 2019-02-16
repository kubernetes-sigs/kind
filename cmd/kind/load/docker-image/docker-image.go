/*
Copyright 2019 The Kubernetes Authors.

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

// Package load implements the `load` command
package load

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	clusternodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/docker"
)

type flagpole struct {
	Name  string
	Nodes []string
}

// NewCommand returns a new cobra.Command for loading an image into a cluster
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("name of image is required")
			}
			return nil
		},
		Use:   "docker-image",
		Short: "loads docker image from host into nodes",
		Long:  "loads docker image from host into all or specified nodes by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.Name,
		"name",
		cluster.DefaultName,
		"the cluster context name",
	)
	cmd.Flags().StringSliceVar(
		&flags.Nodes,
		"nodes",
		nil,
		"comma seperated list of nodes to load images into",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	imageTarName := "image.tar"
	destinationImageTar := fmt.Sprintf("/%s", imageTarName)

	// Get the image into a tar
	err := docker.Save(args[0], imageTarName)
	if err != nil {
		return err
	}

	// List nodes by cluster context name
	n, err := clusternodes.ListByCluster()
	if err != nil {
		return err
	}
	nodes, known := n[flags.Name]
	if !known {
		return errors.Errorf("unknown cluster %q", flags.Name)
	}

	for _, node := range nodes {
		// Copy image tar to each node
		if err := node.CopyTo(imageTarName, destinationImageTar); err != nil {
			return errors.Wrap(err, "failed to copy image to node")
		}

		// Load image into each node
		if err := node.Command(
			"docker", "load", "--input", destinationImageTar,
		).Run(); err != nil {
			return errors.Wrap(err, "failed to load image")
		}
	}
	return nil
}
