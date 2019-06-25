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
	"os"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha3"
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
	utilruntime.Must(v1alpha3.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha3.SchemeGroupVersion))
}

// Load reads the file at path and attempts to convert into a `kind` Config; the file
// can be one of the different API versions defined in scheme.
// If path == "" then the default config is returned
// If path == "-" then reads from stdin
func Load(path string) (*config.Cluster, error) {
	var latestPublicConfig = &v1alpha3.Cluster{}

	if path != "" {
		var err error
		var contents []byte

		if path == "-" {
			// read in stdin
			contents, err = ioutil.ReadAll(os.Stdin)
		} else {
			// read in file
			contents, err = ioutil.ReadFile(path)
		}
		if err != nil {
			return nil, err
		}

		// decode data into a internal api Config object because
		// to leverage on conversion functions for all the api versions
		var cfg = &config.Cluster{}
		err = runtime.DecodeInto(Codecs.UniversalDecoder(), contents, cfg)
		if err != nil {
			return nil, errors.Wrap(err, "decoding failure")
		}

		// converts back to the latest API version to apply defaults
		Scheme.Convert(cfg, latestPublicConfig, nil)
	}

	// apply defaults
	Scheme.Default(latestPublicConfig)

	// converts to internal config
	var cfg = &config.Cluster{}
	Scheme.Convert(latestPublicConfig, cfg, nil)

	// unmarshal the file content into a `kind` Config
	return cfg, nil
}
