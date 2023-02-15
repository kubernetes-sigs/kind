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

// Package validations implements the validation action
package validation

import (
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/createworker"
)

type action struct {
	descriptorPath string
	secretsPath    string
	vaultPassword  string
}

var validator Validator

func NewAction(descriptorPath string, secretsPath string, vaultPassword string) actions.Action {
	return &action{
		descriptorPath: descriptorPath,
		secretsPath:    secretsPath,
		vaultPassword:  vaultPassword,
	}
}

func (a *action) Execute(ctx *actions.ActionContext) error {

	descriptorFile, err := cluster.GetClusterDescriptor(a.descriptorPath)
	if err != nil {
		return err
	}
	secretsFile, err := createworker.GetSecretsFile(a.secretsPath, a.vaultPassword)

	infraProvider := descriptorFile.InfraProvider
	managed := descriptorFile.ControlPlane.Managed

	managedStr := "managed"
	if managed {
		managedStr = "unmanaged"
	}
	ctx.Status.Start("Validating " + infraProvider + " " + managedStr + " descriptor file: " + a.descriptorPath + " and secrets file " + a.secretsPath)
	defer ctx.Status.End(false)

	validator, err := getValidator(infraProvider, managed)
	if err != nil {
		return err
	}
	validator.descriptorFile(*descriptorFile)
	validator.secretsFile(*secretsFile)
	err = validator.validate()

	if err != nil {
		return err
	}

	ctx.Status.End(true)

	return nil

}
