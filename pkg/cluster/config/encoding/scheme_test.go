/*
Copyright 2018 The Kubernetes Authors.

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

package encoding

import (
	"io/ioutil"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSplitYAMLDocuments(t *testing.T) {
	cases := []struct {
		TestName        string
		Path            string
		ExpectDocuments []schema.GroupVersionKind
		ExpectError     bool
	}{
		{
			TestName: "One yaml document",
			Path:     "./testdata/v1alpha1/valid-minimal.yaml",
			ExpectDocuments: []schema.GroupVersionKind{
				{Group: "kind.sigs.k8s.io", Version: "v1alpha1", Kind: "Config"},
			},
			ExpectError: false,
		},
		{
			TestName: "Two yaml documents",
			Path:     "./testdata/v1alpha2/valid-minimal-two-nodes.yaml",
			ExpectDocuments: []schema.GroupVersionKind{
				{Group: "kind.sigs.k8s.io", Version: "v1alpha2", Kind: "Node"},
				{Group: "kind.sigs.k8s.io", Version: "v1alpha2", Kind: "Node"},
			},
			ExpectError: false,
		},
		{
			TestName:    "No kind is specified",
			Path:        "./testdata/invalid-no-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "No apiVersion is specified",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid apiversion",
			Path:        "./testdata/invalid-apiversion.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid kind",
			Path:        "./testdata/invalid-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid yaml",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			contents, err := ioutil.ReadFile(c.Path)
			if err != nil {
				t2.Fatalf("unexpected error while reading test file: %v", err)
			}

			yamlDocuments, err := splitYAMLDocuments(contents)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t2.Fatalf("unexpected error while adding Nodes: %v", err)
				}
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t2.Fatalf("unexpected lack or error while adding Nodes")
				}
			}

			// checks that all the expected yamlDocuments are there
			if len(c.ExpectDocuments) != len(yamlDocuments) {
				t2.Errorf("expected %d documents, saw %d", len(c.ExpectDocuments), len(yamlDocuments))
			}

			// checks that GroupVersionKind for each yamlDocuments
			for i, expectDocument := range c.ExpectDocuments {
				if !reflect.DeepEqual(expectDocument, yamlDocuments[i].GroupVersionKind) {
					t2.Errorf("Invalid document in position %d: expected %v, saw: %v", i, expectDocument, yamlDocuments[i].GroupVersionKind)
				}
			}
		})
	}
}

func TestLoadCurrent(t *testing.T) {
	cases := []struct {
		TestName    string
		Path        string
		ExpectNodes []string
		ExpectError bool
	}{
		{
			TestName:    "no config",
			Path:        "",
			ExpectNodes: []string{"control-plane"}, // no config (empty config path) should return a single node cluster
			ExpectError: false,
		},
		{
			TestName:    "v1alpha1 minimal",
			Path:        "./testdata/v1alpha1/valid-minimal.yaml",
			ExpectNodes: []string{"control-plane"},
			ExpectError: false,
		},
		{
			TestName:    "v1alpha1 with lifecyclehooks",
			Path:        "./testdata/v1alpha1/valid-with-lifecyclehooks.yaml",
			ExpectNodes: []string{"control-plane"},
			ExpectError: false,
		},
		{
			TestName:    "v1alpha1 with more than one doc",
			Path:        "./testdata/v1alpha1/invalid-minimal-two-nodes.yaml",
			ExpectError: true,
		},
		{
			TestName:    "v1alpha2 minimal",
			Path:        "./testdata/v1alpha2/valid-minimal.yaml",
			ExpectNodes: []string{"control-plane"},
			ExpectError: false,
		},
		{
			TestName:    "v1alpha2 lifecyclehooks",
			Path:        "./testdata/v1alpha2/valid-with-lifecyclehooks.yaml",
			ExpectNodes: []string{"control-plane"},
			ExpectError: false,
		},
		{
			TestName:    "v1alpha2 config with 2 nodes",
			Path:        "./testdata/v1alpha2/valid-minimal-two-nodes.yaml",
			ExpectNodes: []string{"control-plane", "worker"},
			ExpectError: false,
		},
		{
			TestName:    "v1alpha2 full HA",
			Path:        "./testdata/v1alpha2/valid-full-ha.yaml",
			ExpectNodes: []string{"etcd", "lb", "control-plane1", "control-plane2", "control-plane3", "worker1", "worker2"},
			ExpectError: false,
		},
		{
			TestName:    "invalid path",
			Path:        "./testdata/not-a-file.bogus",
			ExpectError: true,
		},
		{
			TestName:    "No kind is specified",
			Path:        "./testdata/invalid-no-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "No apiVersion is specified",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid apiversion",
			Path:        "./testdata/invalid-apiversion.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid kind",
			Path:        "./testdata/invalid-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid yaml",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			cfg, err := Load(c.Path)

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t2.Fatalf("unexpected error while unmarshalConfig: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t2.Fatalf("unexpected lack or error while unmarshalConfig")
				}
			}

			if len(cfg.Nodes()) != len(c.ExpectNodes) {
				t2.Errorf("expected %d nodes, saw %d", len(c.ExpectNodes), len(cfg.Nodes()))
			}

			for i, name := range c.ExpectNodes {
				if cfg.Nodes()[i].Name != name {
					t2.Errorf("expected %q node at position %d, saw %q", name, i, cfg.Nodes()[i].Name)
				}
			}
		})
	}
}
