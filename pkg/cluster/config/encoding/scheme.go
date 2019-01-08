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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha1"
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha2"
)

// Scheme is the runtime.Scheme to which all `kind` config API versions and types are registered.
var Scheme = runtime.NewScheme()

// Codecs provides access to encoding and decoding for the scheme.
var Codecs = serializer.NewCodecFactory(Scheme)

func init() {
	AddToScheme(Scheme)
}

// AddToScheme builds the scheme using all known `kind` API versions.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(config.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha2.SchemeGroupVersion))
}

// Load reads the file at path and attempts to convert into a `kind` Config; the file
// can be one of the different API versions defined in scheme.
// If path == "" then the default config is returned
func Load(path string) (*config.Config, error) {
	if path == "" {
		return newDefaultedConfig(), nil
	}

	// read in file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// unmarshal the file content into a `kind` Config
	return unmarshalConfig(contents)
}

// newDefaultedConfig creates a new, defaulted `kind` Config
// with one control-plane node.
func newDefaultedConfig() *config.Config {
	var cfg = &v1alpha2.Node{Role: v1alpha2.ControlPlaneRole}

	// apply defaults
	Scheme.Default(cfg)

	// converts to internal node
	var internalNode = &config.Node{}
	Scheme.Convert(cfg, internalNode, nil)

	// creates the internal cfg and add the node
	var internalCfg = &config.Config{}
	internalCfg.Add(internalNode)

	return internalCfg
}

// yamlDocument identifies a yaml document contained in a yaml file
// by its own GroupVersionKind
type yamlDocument struct {
	GroupVersionKind schema.GroupVersionKind
	Contents         []byte
}

// unmarshalConfig attempt to decode data into a `kind` Config; data can be
// one of the different API versions defined in the Scheme; for v1alpha2
// multiple yaml documents within the same yaml file are supported
func unmarshalConfig(contents []byte) (*config.Config, error) {
	// parses yamlDocuments separated by --- directives
	yamlDocuments, err := splitYAMLDocuments(contents)
	if err != nil {
		return nil, err
	}

	// checks if v1alpha1 config is present in the yamlDocuments
	var v1alpha1Config = false
	for _, doc := range yamlDocuments {
		if doc.GroupVersionKind.GroupVersion() == v1alpha1.SchemeGroupVersion {
			v1alpha1Config = true
		}
	}

	// if using v1alpha1Config,
	if v1alpha1Config {
		// only one yaml document is supported
		if len(yamlDocuments) > 1 {
			return nil, fmt.Errorf("if using v1alpha1 config, only one yaml document should be provided")
		}

		// decode the yaml into a config object
		var v1alpha1Cfg = &v1alpha1.Config{}
		_, _, err := Codecs.UniversalDecoder().Decode(yamlDocuments[0].Contents, nil, v1alpha1Cfg)
		if err != nil {
			return nil, errors.Wrap(err, "decoding failure")
		}

		// apply defaults
		Scheme.Default(v1alpha1Cfg)

		// converts to internal cfg. this will give Config with a single node with control-plane role
		// NB. we are using custom conversion due to the complexity of conversion from v1alpha1 config
		var internalCfg = &config.Config{}
		v1alpha1Cfg.Convert(internalCfg)

		return internalCfg, nil
	}

	// Otherwise it is a multi node, v1alpha2 config
	var cfg = &config.Config{}

	// Process all the yamlDocuments
	for _, doc := range yamlDocuments {
		// decode data into a Node object
		var v1alpha2node = &v1alpha2.Node{}
		_, _, err := Codecs.UniversalDecoder().Decode(doc.Contents, nil, v1alpha2node)
		if err != nil {
			return nil, errors.Wrap(err, "decoding failure")
		}

		// apply defaults
		Scheme.Default(v1alpha2node)

		// converts to internal cfg
		var internalNode = &config.Node{}
		Scheme.Convert(v1alpha2node, internalNode, nil)

		// adds the node to the internal config
		// in case replicas shuold be generated for the nodes, Add will takes care.
		if err := cfg.Add(internalNode); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// splitYAMLDocuments divides a yaml file into yamlDocuments,
// and identifies each document with its own GroupVersionKind
func splitYAMLDocuments(contents []byte) (yamlDocuments []yamlDocument, err error) {
	yamlDocuments = []yamlDocument{}

	// parses yamlDocuments separated by --- directives
	buf := bytes.NewBuffer(contents)
	reader := utilyaml.NewYAMLReader(bufio.NewReader(buf))
	for {
		// Read one YAML document at a time, until io.EOF is returned
		documentContents, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if len(documentContents) == 0 {
			break
		}

		// Unmarshal the TypeMeta information of this byte slice
		typeMetaInfo := runtime.TypeMeta{}
		if err := yaml.Unmarshal(documentContents, &typeMetaInfo); err != nil {
			return nil, err
		}

		// Require TypeMeta information to be present
		if len(typeMetaInfo.APIVersion) == 0 || len(typeMetaInfo.Kind) == 0 {
			return nil, errors.New("kind and apiVersion are mandatory information that needs to be specified in all YAML documents")
		}

		// Build a GroupVersionKind object from the TypeMeta object
		documentGroupVersion, err := schema.ParseGroupVersion(typeMetaInfo.APIVersion)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse apiVersion")
		}
		documentGroupVersionKind := documentGroupVersion.WithKind(typeMetaInfo.Kind)

		// checks that the Kind is a known type
		if !Scheme.Recognizes(documentGroupVersionKind) {
			return nil, errors.Errorf("unknown %q object type", documentGroupVersionKind)
		}

		// add the document to the list of documents
		yamlDocuments = append(yamlDocuments, yamlDocument{GroupVersionKind: documentGroupVersionKind, Contents: documentContents})
	}

	return yamlDocuments, nil
}
