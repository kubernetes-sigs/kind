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

// Package images implements the `images` command
package images

import (
	"fmt"

	"github.com/spf13/cobra"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of nodes for a given cluster
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "images",
		Short: "lists official kind node images by tag",
		Long:  "lists official kind node images by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags)
		},
	}
	return cmd
}

func runE(flags *flagpole) error {
	// List images available in dockerhub.
	image := "kindest/node"
	i, err := retrieveFromDockerHub(image)
	if err != nil {
		return err
	}
	for _, tag := range i {
		fmt.Printf("%s:%s\n", image, tag)
	}
	return nil
}
