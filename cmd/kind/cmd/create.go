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

package cmd

import (
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/pkg/cluster"
)

func newCreateCommand() *cobra.Command {
	return &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "create",
		Short: "Creates a cluster",
		Long:  "Creates a Kubernetes cluster",
		Run:   runCreate,
	}
}

func runCreate(cmd *cobra.Command, args []string) {
	// TODO(bentheelder): make this configurable
	config := cluster.NewConfig("")
	ctx := cluster.NewContext(config)
	err := ctx.Create()
	if err != nil {
		glog.Errorf("Failed to create cluster: %v", err)
		os.Exit(-1)
	}
}
