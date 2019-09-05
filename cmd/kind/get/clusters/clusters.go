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

// Package clusters implements the `clusters` command
package clusters

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster/nodes"

	"k8s.io/kubernetes/pkg/printers"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "clusters",
		Short: "lists existing kind clusters by their name",
		Long:  "lists existing kind clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE()
		},
	}
	return cmd
}

func runE() error {
	clusters, err := nodes.ListByCluster()
	if err != nil {
		return err
	}
	printer := printers.GetNewTabWriter(os.Stdout)
	fmt.Fprintln(printer, strings.Join([]string{"NAME", "RUNNING", "VERSION"}, "\t"))
	for cluster, nodeList := range clusters {
		size := strconv.Itoa(len(nodeList))
		running := 0
		version := ""
		for _, n := range nodeList {
			status, _ := n.IsRunning()
			if status {
				running++
			}
			v, err := n.KubeVersion()
			if err == nil {
				version = v
			}
		}
		runState := strconv.Itoa(running) + "/" + size
		fmt.Fprintln(printer, strings.Join([]string{cluster, runState, version}, "\t"))
	}
	printer.Flush()
	return nil
}
