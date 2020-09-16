/*
Copyright 2020 The Kubernetes Authors.

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

// Package completion implements the `default` command
package defaultconfig

import (
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
	v1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

// NewCommand returns a new cobra.Command for default config
func NewCommand(_ log.Logger, streams cmd.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "default",
		Short: "Output default config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return defaultConfig(streams)
		},
	}
	return cmd
}

func defaultConfig(streams cmd.IOStreams) error {
	cluster := v1alpha4.Cluster{
		TypeMeta: v1alpha4.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "kind.x-k8s.io/v1alpha4",
		},
	}

	v1alpha4.SetDefaultsCluster(&cluster)

	encoder := yaml.NewEncoder(streams.Out)
	encoder.SetIndent(2)
	return encoder.Encode(&cluster)
}
