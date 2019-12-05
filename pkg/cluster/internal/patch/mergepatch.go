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
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"
)

type mergePatch struct {
	raw       string    // the original raw data
	json      []byte    // the processed data (in JSON form)
	matchInfo matchInfo // for matching resources
}

func parseMergePatches(rawPatches []string) ([]mergePatch, error) {
	patches := []mergePatch{}
	for _, raw := range rawPatches {
		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json, err := yaml.YAMLToJSON([]byte(raw))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patches = append(patches, mergePatch{
			raw:       raw,
			json:      json,
			matchInfo: matchInfo,
		})
	}
	return patches, nil
}
