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

// NewConfig returns the default config according to requested number of control-plane
// and worker nodes
func NewConfig(controlPlanes, workers int32) (*config.Config, error) {
	var latestPublicConfig = &v1alpha2.Config{}

	// create default config according to requested number of control-plane and worker nodes
	// adds the control-plane node(s)
	latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, v1alpha2.Node{Role: v1alpha2.ControlPlaneRole, Replicas: &controlPlanes})
	// if more than one control-plane node(s), automatically add an external load balancer
	if controlPlanes > 1 {
		latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, v1alpha2.Node{Role: v1alpha2.ExternalLoadBalancerRole})
	}

	// adds the worker node(s), if any
	if workers > 0 {
		latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, v1alpha2.Node{Role: v1alpha2.WorkerRole, Replicas: &workers})
	}

	// apply defaults
	Scheme.Default(latestPublicConfig)

	// converts to internal config
	var cfg = &config.Config{}
	Scheme.Convert(latestPublicConfig, cfg, nil)

	// unmarshal the file content into a `kind` Config
	return cfg, nil
}

// Load reads the file at path and attempts to convert into a `kind` Config; the file
// can be one of the different API versions defined in scheme.
func Load(path string) (*config.Config, error) {
	// override default config by reading a file
	latestPublicConfig := &v1alpha2.Config{}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// decode data into a internal api Config object because
	// to leverage on conversion functions for all the api versions
	var filecfg = &config.Config{}
	err = runtime.DecodeInto(Codecs.UniversalDecoder(), contents, filecfg)
	if err != nil {
		return nil, errors.Wrap(err, "decoding failure")
	}

	// converts back to the latest API version to apply defaults
	Scheme.Convert(filecfg, latestPublicConfig, nil)

	// apply defaults
	Scheme.Default(latestPublicConfig)

	// converts to internal config
	var cfg = &config.Config{}
	Scheme.Convert(latestPublicConfig, cfg, nil)

	// unmarshal the file content into a `kind` Config
	return cfg, nil
}
