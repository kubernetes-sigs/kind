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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	clusternodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/fs"
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
		"comma separated list of nodes to load images into",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// List nodes by cluster context name
	n, err := clusternodes.ListByCluster()
	if err != nil {
		return err
	}
	nodes, known := n[flags.Name]
	if !known {
		return errors.Errorf("unknown cluster %q", flags.Name)
	}

	// map cluster nodes by their name
	nodesByName := map[string]clusternodes.Node{}
	for _, node := range nodes {
		// TODO(bentheelder): this depends on the fact that ListByCluster()
		// will have name for nameOrId.
		nodesByName[node.String()] = node
	}

	// pick only the user selected nodes and ensure they exist
	// the default is all nodes unless flags.Nodes is set
	selectedNodes := nodes
	if len(flags.Nodes) > 0 {
		selectedNodes := []clusternodes.Node{}
		for _, name := range flags.Nodes {
			node, ok := nodesByName[name]
			if !ok {
				return errors.Errorf("unknown node: %s", name)
			}
			selectedNodes = append(selectedNodes, node)
		}
	}

	// Save the image into a tar
	dir, err := fs.TempDir("", "image-tar")
	if err != nil {
		return errors.Wrap(err, "failed to create tempdir")
	}
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = docker.Save(args[0], imageTarPath)
	if err != nil {
		return err
	}

	// Load the image into every node
	for _, node := range selectedNodes {
		if err := loadImage(imageTarPath, &node); err != nil {
			return err
		}
	}
	return nil
}

func loadImage(imageTarName string, node *clusternodes.Node) error {
	// Copy image tar to each node
	f, err := os.Open(imageTarName)
	if err != nil {
		return errors.Wrap(err, "failed to open image")
	}
	defer f.Close()

	// Load image into each node
	cmd := node.Command(
		"docker", "load",
	)
	cmd.SetStdin(f)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to load image")
	}
	return nil
}
