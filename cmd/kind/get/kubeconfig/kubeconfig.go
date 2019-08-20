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

// Package kubeconfig implements the `kubeconfig` command
package kubeconfig

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
)

type flagpole struct {
	Name     string
	Internal bool
}

// NewCommand returns a new cobra.Command for getting the kubeconfig with the internal node IP address
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig",
		Short: "prints cluster kubeconfig",
		Long:  "prints cluster kubeconfig",
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
	cmd.Flags().BoolVar(
		&flags.Internal,
		"internal",
		false,
		"use internal address instead of external",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// Check if the cluster name exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if !known {
		return errors.Errorf("unknown cluster %q", flags.Name)
	}
	ctx := cluster.NewContext(flags.Name)
	if flags.Internal {
		bytes, err := ctx.KubeConfig()
		if err != nil {
			return errors.Wrap(err, "failed to get cluster internal kubeconfig")
		}
		_, _ = os.Stdout.Write(bytes)
		return nil
	}

	f, err := os.Open(ctx.KubeConfigPath())
	if err != nil {
		return errors.Wrap(err, "failed to get cluster kubeconfig")
	}
	defer f.Close()
	if _, err := io.Copy(os.Stdout, f); err != nil {
		return errors.Wrap(err, "failed to copy kubeconfig")
	}

	return nil
}
