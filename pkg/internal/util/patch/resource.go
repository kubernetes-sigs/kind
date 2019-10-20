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

package patch

import (
	"io"
	"io/ioutil"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"

	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"
)

type resource struct {
	raw       string    // the original raw data
	json      []byte    // the processed data (in JSON form), may be mutated
	matchInfo matchInfo // for matching patches
}

func (r *resource) apply6902Patch(patch json6902Patch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := patch.patch.Apply(r.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r *resource) applyMergePatch(patch mergePatch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := jsonpatch.MergePatch(r.json, patch.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r resource) matches(o matchInfo) bool {
	m := &r.matchInfo
	// we require kind to match, but if the patch does not specify
	// APIVersion we ignore it (eg to allow trivial patches across kubeadm versions)
	return m.Kind == o.Kind && (o.APIVersion == "" || m.APIVersion == o.APIVersion)
}

func (r *resource) encodeTo(w io.Writer) error {
	encoded, err := yaml.JSONToYAML(r.json)
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := w.Write(encoded); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func parseResources(yamlDocumentStream string) ([]resource, error) {
	resources := []resource{}
	documents, err := splitYAMLDocuments(yamlDocumentStream)
	if err != nil {
		return nil, err
	}
	for _, raw := range documents {
		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json, err := yaml.YAMLToJSON([]byte(raw))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		resources = append(resources, resource{
			raw:       raw,
			json:      json,
			matchInfo: matchInfo,
		})
	}
	return resources, nil
}

func splitYAMLDocuments(yamlDocumentStream string) ([]string, error) {
	documents := []string{}
	buff := make([]byte, len(yamlDocumentStream))
	r := yamlutil.NewDocumentDecoder(ioutil.NopCloser(strings.NewReader(yamlDocumentStream)))
	for {
		n, err := r.Read(buff)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Wrap(err, "error splitting documents")
		}
		documents = append(documents, string(buff[:n]))
	}
	return documents, nil
}
