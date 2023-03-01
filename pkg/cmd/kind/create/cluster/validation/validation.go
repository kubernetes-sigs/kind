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
	"os"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kind/pkg/commons"
)

var validator Validator

func InitValidator(descriptorPath string) error {
	descriptorFile, err := commons.GetClusterDescriptor(descriptorPath)
	if err != nil {
		return err
	}

	infraProvider := descriptorFile.InfraProvider
	managed := descriptorFile.ControlPlane.Managed
	validator, err = getValidator(infraProvider, managed)
	if err != nil {
		return err
	}

	validator.DescriptorFile(*descriptorFile)
	return nil
}

func ExecuteDescriptorValidations() error {

	if err := validator.Validate("descriptor"); err != nil {
		return err
	}
	return nil
}

func ExecuteSecretsValidations(secretsPath string, vaultPassword string) error {
	_, err := os.Stat("./secrets.yml")
	if err == nil {
		secretsFile, err := commons.GetSecretsFile(secretsPath, vaultPassword)
		if err != nil {
			return err
		}

		validator.SecretsFile(*secretsFile)
		err = validator.Validate("secrets")
		if err != nil {
			return err
		}
	}

	return nil
}

func getClusterDescriptor(filename string) (*map[string]interface{}, error) {
	fileRAW, err := os.ReadFile("./" + filename)
	if err != nil {
		return nil, err
	}
	file := make(map[string]interface{})
	err = yaml.Unmarshal(fileRAW, &file)
	if err != nil {
		return nil, err
	}

	return &file, nil
}
