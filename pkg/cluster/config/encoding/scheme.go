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

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha1"
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
	utilruntime.Must(scheme.SetVersionPriority(v1alpha1.SchemeGroupVersion))
}

// newDefaultedConfig creates a new, defaulted `kind` Config
// Defaulting uses Scheme registered defaulting functions
func newDefaultedConfig() *config.Config {
	var cfg = &v1alpha1.Config{}

	// apply defaults
	Scheme.Default(cfg)

	// converts to internal cfg
	var internalCfg = &config.Config{}
	Scheme.Convert(cfg, internalCfg, nil)

	return internalCfg
}

// unmarshalConfig attempt to decode data into a `kind` Config; data can be
// one of the different API versions defined in the Scheme.
func unmarshalConfig(data []byte) (*config.Config, error) {
	var cfg = &v1alpha1.Config{}

	// decode data into a config object
	_, _, err := Codecs.UniversalDecoder().Decode(data, nil, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "decoding failure")
	}

	// apply defaults
	Scheme.Default(cfg)

	// converts to internal cfg
	var internalCfg = &config.Config{}
	Scheme.Convert(cfg, internalCfg, nil)

	return internalCfg, nil
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
