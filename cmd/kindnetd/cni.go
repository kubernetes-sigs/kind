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

package main

import (
	"io"
	"os"
	"reflect"
	"text/template"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
)

/* cni config management */

// CNIConfigInputs is supplied to the CNI config template
type CNIConfigInputs struct {
	PodCIDR string
}

// ComputeCNIConfigInputs computes the template inputs for CNIConfigWriter
func ComputeCNIConfigInputs(node corev1.Node) CNIConfigInputs {
	podCIDR := node.Spec.PodCIDR
	return CNIConfigInputs{
		PodCIDR: podCIDR,
	}
}

// CNIConfigPath is where kindnetd will write the computed CNI config
const CNIConfigPath = "/etc/cni/net.d/10-kindnet.conflist"

// CNIConfigWriter no-ops re-writing config with the same inputs
// NOTE: should only be called from a single goroutine
type CNIConfigWriter struct {
	path       string
	template   string
	lastInputs CNIConfigInputs
}

// Write will write the config based on
func (c *CNIConfigWriter) Write(inputs CNIConfigInputs) error {
	if reflect.DeepEqual(inputs, c.lastInputs) {
		return nil
	}

	// use an extension not recognized by CNI to write the contents initially
	// https://github.com/containerd/go-cni/blob/891c2a41e18144b2d7921f971d6c9789a68046b2/opts.go#L170
	// then we can rename to atomically make the file appear
	f, err := os.Create(c.path + ".temp")
	if err != nil {
		return err
	}

	// actually write the config
	if err := writeCNIConfig(f, c.template, inputs); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Sync()
	f.Close()

	// then we can rename to the target config path
	if err := os.Rename(f.Name(), c.path); err != nil {
		return err
	}

	// we're safely done now, record the inputs
	c.lastInputs = inputs
	return nil
}

func writeCNIConfig(w io.Writer, rawTemplate string, data CNIConfigInputs) error {
	t, err := template.New("cni-json").Parse(rawTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse cni template")
	}
	return t.Execute(w, &data)
}
