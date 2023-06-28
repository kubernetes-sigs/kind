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

package createworker

import (
	"embed"
	_ "embed"

	"io/ioutil"
	"os"

	"path/filepath"

	"time"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

//go:embed files/*/internal-ingress-nginx.yaml
var internalIngressFiles embed.FS

func override_vars(descriptorFile commons.DescriptorFile, credentialsMap map[string]string, ctx *actions.ActionContext, infra *Infra, provider Provider) error {

	override_vars, err := infra.getOverrideVars(descriptorFile, credentialsMap) //OverrideVars: {"overrideFileName1": {"overrideVariable1": "overrideValue1", ...}}
	if err != nil {
		return err
	}

	if len(override_vars) > 0 {
		ctx.Status.Start("Generating override_vars structure ⚒️")
		defer ctx.Status.End(false)
		for filename, overrideValue := range override_vars {
			overrideVarsDir := "override_vars"
			originalFilePath := filepath.Join(overrideVarsDir, filename)
			err := createBackupOverrideVars(originalFilePath)
			if err != nil {
				return err
			}

			err = os.MkdirAll(filepath.Dir(originalFilePath), os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "error creating override_vars directory")
			}

			err = ioutil.WriteFile(originalFilePath, overrideValue, os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "error writing corresponding '"+originalFilePath+"'")
			}

		}
		ctx.Status.End(true) // End Generating override_vars structure

	}

	return nil
}

func internalNginxOverrideVars() error {
	err := createBackupOverrideVars("ingress-nginx.yaml")
	if err != nil {
		return err
	}
	return nil
}

func createBackupOverrideVars(originalFilePath string) error {
	timestamp := time.Now().Format("20060102150405")
	newFilePath := filepath.Dir(originalFilePath) + "." + timestamp

	// Make a backup of existing override_vars
	_, err := os.Stat(originalFilePath)
	if err == nil {
		err := os.Rename(filepath.Dir(originalFilePath), newFilePath)
		if err != nil {
			return errors.Wrap(err, "error renaming original override_vars directory")
		}
	}
	return nil
}

func addOverrideVar(path string, value []byte, ov map[string][]byte) map[string][]byte {
	if path != "" && string(value) != "" {
		ov[path] = value
	}
	return ov
}
