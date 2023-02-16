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
	"fmt"
	"os"

	vault "github.com/sosedoff/ansible-vault-go"
	"gopkg.in/yaml.v2"
)

type action struct {
	descriptorPath string
	secretsPath    string
	vaultPassword  string
}

var validator Validator

// func NewAction(descriptorPath string, secretsPath string, vaultPassword string) actions.Action {
// 	return &action{
// 		descriptorPath: descriptorPath,
// 		secretsPath:    secretsPath,
// 		vaultPassword:  vaultPassword,
// 	}
// }

func ExecuteDescriptorValidations(descriptorPath string) (*string, *bool, error) {
	descriptorFile, err := getClusterDescriptor(descriptorPath)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("validation descriptor: ")
	//fmt.Println(descriptorFile)
	fmt.Println((*descriptorFile)["control_plane"].(map[interface{}]interface{})["managed"].(bool))

	infraProvider := (*descriptorFile)["infra_provider"].(string)
	managed := (*descriptorFile)["control_plane"].(map[interface{}]interface{})["managed"].(bool)

	validator, err := getValidator(infraProvider, managed)
	if err != nil {
		return nil, nil, err
	}
	validator.descriptorFile(*descriptorFile)
	err = validator.validate("descriptor")

	if err != nil {
		return nil, nil, err
	}

	return &infraProvider, &managed, nil
}

func ExecuteSecretsValidations(infra string, managed bool, secretsPath string, vaultPassword string) error {
	secretsFile, err := GetSecretsFile(secretsPath, vaultPassword)
	if err != nil {
		return err
	}
	validator, err := getValidator(infra, managed)
	if err != nil {
		return err
	}
	validator.secretsFile(*secretsFile)
	err = validator.validate("secrets")
	if err != nil {
		return err
	}
	return nil
}

// func ExecuteValidations2(descriptorPath string, secretsPath string, vaultPassword string) error {
// 	//func (a *action) Execute(ctx *actions.ActionContext) error {

// 	descriptorFile, err := cluster.GetClusterDescriptor(descriptorPath)
// 	if err != nil {
// 		return err
// 	}
// 	secretsFile, err := createworker.GetSecretsFile(secretsPath, vaultPassword)

// 	infraProvider := descriptorFile.InfraProvider
// 	managed := descriptorFile.ControlPlane.Managed

// 	//managedStr := "managed"
// 	// if managed {
// 	// 	managedStr = "unmanaged"
// 	// }
// 	//ctx.Status.Start("Validating " + infraProvider + " " + managedStr + " descriptor file: " + a.descriptorPath + " and secrets file: " + a.secretsPath)
// 	//defer ctx.Status.End(false)

// 	validator, err := getValidator(infraProvider, managed)
// 	if err != nil {
// 		return err
// 	}
// 	validator.descriptorFile(*descriptorFile)
// 	validator.secretsFile(*secretsFile)
// 	err = validator.validate()

// 	if err != nil {
// 		return err
// 	}

// 	//ctx.Status.End(true)

// 	return nil

// }

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

	// validate := val.New()
	// err = validate.Struct(descriptorFile)
	// if err != nil {
	// 	return nil, err
	// }
	return &file, nil
}

func decryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return data, nil
}

func GetSecretsFile(secretsPath string, vaultPassword string) (*map[string]interface{}, error) {
	secretRaw, err := decryptFile(secretsPath, vaultPassword)
	file := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(secretRaw), &file)
	if err != nil {
		return nil, err
	}

	// validate := val.New()
	// err = validate.Struct(descriptorFile)
	// if err != nil {
	// 	return nil, err
	// }
	return &file, nil
}
