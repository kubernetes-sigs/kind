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

func override_vars(descriptorFile commons.DescriptorFile, ctx *actions.ActionContext, infra *Infra) error {

	overrideVarsDir := "override_vars"
	originalFilePath := filepath.Join(overrideVarsDir, "ingress-nginx.yaml")
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

	requiredInternalNginx, err := infra.internalNginx(descriptorFile.Networks)
	if err != nil {
		return err
	}
	if requiredInternalNginx {

		ctx.Status.Start("Generating override_vars structure ⚒️")
		defer ctx.Status.End(false)

		err = os.MkdirAll(filepath.Dir(originalFilePath), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "error creating override_vars directory")
		}

		// Create ingress-nginx.yaml in override_vars folder if required
		internalIngressFilePath := "files/" + descriptorFile.InfraProvider + "/internal-ingress-nginx.yaml"
		internalIngressFile, err := internalIngressFiles.Open(internalIngressFilePath)
		if err != nil {
			return errors.Wrap(err, "error opening the internal ingress nginx file")
		}
		defer internalIngressFile.Close()

		internalIngressContent, err := ioutil.ReadAll(internalIngressFile)
		if err != nil {
			return errors.Wrap(err, "error reading the internal ingress nginx file")
		}
		err = ioutil.WriteFile(originalFilePath, []byte(internalIngressContent), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "error writing corresponding 'override_vars/ingress-nginx.yaml'")
		}

		ctx.Status.End(true) // End Generating override_vars structure
	}

	return nil
}
