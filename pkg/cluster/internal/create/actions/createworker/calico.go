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
	"bytes"
	"embed"
	"text/template"

	"sigs.k8s.io/kind/pkg/commons"
)

//go:embed templates/*
var ctel embed.FS

func getCalicoManifest(descriptorFile commons.DescriptorFile) (string, error) {

	var tpl bytes.Buffer
	var helmValuesFilename string = "calico-helm-values.tmpl"
	t, err := template.New("").ParseFS(ctel, "templates/"+helmValuesFilename)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, helmValuesFilename, descriptorFile)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}
