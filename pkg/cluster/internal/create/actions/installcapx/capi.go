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

// Package installcapx implements the install CAPI action
package installcapx

import (
	"bytes"
	"os"

	"gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

type action struct{}

// SecretsFile represents the YAML structure in the secrets.yaml file
type SecretsFile struct {
	Secrets struct {
		AWS struct {
			Credentials struct {
				AccessKey string `yaml:"access_key"`
				SecretKey string `yaml:"secret_key"`
				Region    string `yaml:"region"`
				AccountID string `yaml:"account_id"`
			} `yaml:"credentials"`
			B64Credentials string `yaml:"b64_credentials"`
		} `yaml:"aws"`
		GithubToken string `yaml:"github_token"`
	} `yaml:"secrets"`
}

// NewAction returns a new action for installing default CAPI
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {

	ctx.Status.Start("Installing CAPx in local 🎖️")
	defer ctx.Status.End(false)

	err := installCAPA(ctx)
	if err != nil {
		return err
	}

	// mark success
	ctx.Status.End(true) // End Installing CAPx in local

	return nil
}

// installCAPG installs CAPG in the local cluster
func installCAPG(ctx *actions.ActionContext) error {

	ctx.Status.Start("CAPG: ... ")

	defer ctx.Status.End(false)

	return nil
}

// installCAPZ installs CAPZ in the local cluster
func installCAPZ(ctx *actions.ActionContext) error {

	ctx.Status.Start("CAPZ: ... ")

	defer ctx.Status.End(false)

	return nil
}
