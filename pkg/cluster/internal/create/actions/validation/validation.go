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
)

type action struct {
	provider       string
	managed        bool
	descriptorPath string
	secretsPath    string
}

var validator Validator

func NewAction(descriptorPath string, secretsPath string) actions.Action {
	return &action{
		descriptorPath: descriptorPath,
		secretsPath:    secretsPath,
	}
}

func (a *action) Execute(ctx *actions.ActionContext) error {

	descriptorFile, err := cluster.GetClusterDescriptor(a.descriptorPath)
	if err != nil {
		return err
	}

	infraProvider := descriptorFile.InfraProvider
	managed := descriptorFile.ControlPlane.Managed

	managedStr := "managed"
	if managed {
		managedStr = "unmanaged"
	}
	ctx.Status.Start("Validating " + a.provider + " " + managedStr + " descriptor file: " + a.descriptorPath + " and secrets file " + a.secretsPath)
	defer ctx.Status.End(false)

	validator, err := getValidator(infraProvider, managed)
	if err != nil {
		return err
	}
	validator.setDescriptorPath(a.descriptorPath)
	validator.setSecretsPath(a.secretsPath)
	err = validator.validate()

	if err != nil {
		return err
	}

	ctx.Status.End(true)

	return nil

}
