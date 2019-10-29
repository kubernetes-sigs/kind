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

package v1alpha3

import (
	"fmt"
	"strings"
)

/*
Custom YAML (de)serialization for these types
*/

// UnmarshalYAML implements custom decoding YAML
// https://godoc.org/gopkg.in/yaml.v3
func (m *Mount) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// this is basically Mount, except Propagation is a string for further parsing
	type MountYaml struct {
		ContainerPath  string `yaml:"containerPath,omitempty"`
		HostPath       string `yaml:"hostPath,omitempty"`
		Readonly       bool   `yaml:"readOnly,omitempty"`
		SelinuxRelabel bool   `yaml:"selinuxRelabel,omitempty"`
		Propagation    string `yaml:"propagation,omitempty"`
	}
	aux := MountYaml{}
	if err := unmarshal(&aux); err != nil {
		return err
	}
	// copy over normal fields
	m.ContainerPath = aux.ContainerPath
	m.HostPath = aux.HostPath
	m.Readonly = aux.Readonly
	m.SelinuxRelabel = aux.SelinuxRelabel
	// handle special field
	if aux.Propagation != "" {
		val, ok := MountPropagationNameToValue[aux.Propagation]
		if !ok {
			return fmt.Errorf("unknown propagation value: %s", aux.Propagation)
		}
		m.Propagation = MountPropagation(val)
	}
	return nil
}

// UnmarshalYAML implements custom decoding YAML
// https://godoc.org/gopkg.in/yaml.v3
func (p *PortMapping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// this is basically PortMappingYaml, except Protocol is a string for further parsing
	type PortMappingYaml struct {
		ContainerPort int32  `yaml:"containerPort,omitempty"`
		HostPort      int32  `yaml:"hostPort,omitempty"`
		ListenAddress string `yaml:"listenAddress,omitempty"`
		Protocol      string `yaml:"protocol"`
	}
	aux := PortMappingYaml{}
	if err := unmarshal(&aux); err != nil {
		return err
	}
	// copy normal fields
	p.ContainerPort = aux.ContainerPort
	p.HostPort = aux.HostPort
	p.ListenAddress = aux.ListenAddress
	// handle special field
	if aux.Protocol != "" {
		val, ok := PortMappingProtocolNameToValue[strings.ToUpper(aux.Protocol)]
		if !ok {
			return fmt.Errorf("unknown protocol value: %s", aux.Protocol)
		}
		p.Protocol = PortMappingProtocol(val)
	}
	return nil
}
