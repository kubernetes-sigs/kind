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
	"io"
	"io/ioutil"

	"syscall"
	"time"

	term "golang.org/x/term"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd/kind/create/cluster/validation"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/runtime"
)

type flagpole struct {
	Name           string
	Config         string
	ImageName      string
	Retain         bool
	Wait           time.Duration
	Kubeconfig     string
	VaultPassword  string
	DescriptorPath string
	MoveManagement bool
	AvoidCreation  bool
	ForceDelete    bool
}

const clusterDefaultPath = "./cluster.yaml"
const secretsDefaultPath = "./secrets.yml"

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a local Kubernetes cluster",
		Long:  "Creates a local Kubernetes cluster using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.OverrideDefaultName(cmd.Flags())
			return runE(logger, streams, flags)
		},
	}
	cmd.Flags().StringVarP(
		&flags.Name,
		"name",
		"n",
		"",
		"cluster name, overrides KIND_CLUSTER_NAME, config (default kind)",
	)
	cmd.Flags().StringVar(
		&flags.Config,
		"config",
		"",
		"path to a kind config file",
	)
	cmd.Flags().StringVar(
		&flags.ImageName,
		"image",
		"",
		"node docker image to use for booting the cluster",
	)
	cmd.Flags().BoolVar(
		&flags.Retain,
		"retain",
		false,
		"retain nodes for debugging when cluster creation fails",
	)
	cmd.Flags().DurationVar(
		&flags.Wait,
		"wait",
		time.Duration(0),
		"wait for control plane node to be ready (default 0s)",
	)
	cmd.Flags().StringVar(
		&flags.Kubeconfig,
		"kubeconfig",
		"",
		"sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config",
	)
	cmd.Flags().StringVarP(
		&flags.VaultPassword,
		"vault-password",
		"p",
		"",
		"sets vault password to encrypt secrets",
	)
	cmd.Flags().StringVarP(
		&flags.DescriptorPath,
		"descriptor",
		"d",
		"",
		"allows you to indicate the name of the descriptor located in this directory. By default it is cluster.yaml",
	)
	cmd.Flags().BoolVar(
		&flags.MoveManagement,
		"keep-mgmt",
		false,
		"by setting this flag the cluster management will be kept in the kind",
	)
	cmd.Flags().BoolVar(
		&flags.AvoidCreation,
		"avoid-creation",
		false,
		"by setting this flag the worker cluster won't be created",
	)
	cmd.Flags().BoolVar(
		&flags.ForceDelete,
		"delete-previous",
		false,
		"by setting this flag the local cluster will be deleted",
	)

	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams, flags *flagpole) error {

	err := validateFlags(flags)
	if err != nil {
		return err
	}

	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		runtime.GetDefault(logger),
	)

	if flags.DescriptorPath == "" {
		flags.DescriptorPath = clusterDefaultPath
	}
	err = validation.InitValidator(flags.DescriptorPath)
	if err != nil {
		return err
	}

	// handle config flag, we might need to read from stdin
	withConfig, err := configOption(flags.Config, streams.In)
	if err != nil {
		return err
	}

	if flags.VaultPassword == "" {
		flags.VaultPassword, err = setPassword()
		if err != nil {
			return err
		}
	}

	err = validation.ExecuteSecretsValidations(secretsDefaultPath, flags.VaultPassword)
	if err != nil {
		return err
	}

	err = validation.ExecuteDescriptorValidations()
	if err != nil {
		return err
	}

	err = validation.ExecuteCommonsValidations()
	if err != nil {
		return err

	}

	// create the cluster
	if err = provider.Create(
		flags.Name,
		flags.VaultPassword,
		flags.DescriptorPath,
		flags.MoveManagement,
		flags.AvoidCreation,
		withConfig,
		cluster.CreateWithNodeImage(flags.ImageName),
		cluster.CreateWithRetain(flags.Retain),
		cluster.CreateWithMove(flags.MoveManagement),
		cluster.CreateWithAvoidCreation(flags.AvoidCreation),
		cluster.CreateWithForceDelete(flags.ForceDelete),
		cluster.CreateWithWaitForReady(flags.Wait),
		cluster.CreateWithKubeconfigPath(flags.Kubeconfig),
		cluster.CreateWithDisplayUsage(true),
		cluster.CreateWithDisplaySalutation(true),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// configOption converts the raw --config flag value to a cluster creation
// option matching it. it will read from stdin if the flag value is `-`
func configOption(rawConfigFlag string, stdin io.Reader) (cluster.CreateOption, error) {
	// if not - then we are using a real file
	if rawConfigFlag != "-" {
		return cluster.CreateWithConfigFile(rawConfigFlag), nil
	}
	// otherwise read from stdin
	raw, err := ioutil.ReadAll(stdin)
	if err != nil {
		return nil, errors.Wrap(err, "error reading config from stdin")
	}
	return cluster.CreateWithRawConfig(raw), nil
}

func setPassword() (string, error) {
	firstPassword, err := requestPassword("Vault Password: ")
	if err != nil {
		return "", err
	}
	secondPassword, err := requestPassword("Rewrite Vault Password:")
	if err != nil {
		return "", err
	}
	if firstPassword != secondPassword {
		return "", errors.New("The passwords do not match.")
	}

	return firstPassword, nil
}

func requestPassword(request string) (string, error) {
	fmt.Print(request)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Print("\n")
	return string(bytePassword), nil
}

func validateFlags(flags *flagpole) error {
	count := 0
	if flags.AvoidCreation {
		count++
	}
	if flags.Retain {
		count++
	}
	if flags.MoveManagement {
		count++
	}
	if count > 1 {
		return errors.New("Flags --retain, --avoid-creation, and --keep-mgmt are mutually exclusive")
	}
	return nil
}
