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

// Package keosinstaller creates the KEOS descriptor file
package keosinstaller

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// DescriptorFile represents the YAML structure in the cluster.yaml file
type DescriptorFile struct {
	ClusterID string `yaml:"cluster_id"`
	Keos      struct {
		Domain         string `yaml:"domain"`
		ExternalDomain string `yaml:"external_domain"`
		Flavour        string `yaml:"flavour"`
	} `yaml:"keos"`
	K8SVersion string  `yaml:"k8s_version"`
	Bastion    Bastion `yaml:"bastion"`
	Networks   struct {
		VPCID string `yaml:"vpc_id"`
	}
	ExternalRegistry map[string]interface{} `yaml:"external_registry"`
	//      ExternalRegistry     struct {
	//              AuthRequired    bool `yaml: auth_required`
	//              Type            string `yaml: type`
	//              URL             string `yaml: url`
	//      }
	Nodes struct {
		KubeNode struct {
			AmiID string `yaml:"ami_id"`
			Disks []struct {
				DeviceName string `yaml:"device_name"`
				Name       string `yaml:"name"`
				Path       string `yaml:"path,omitempty"`
				Size       int    `yaml:"size"`
				Type       string `yaml:"type"`
				Volumes    []struct {
					Name string `yaml:"name"`
					Path string `yaml:"path"`
					Size string `yaml:"size"`
				} `yaml:"volumes,omitempty"`
			} `yaml:"disks"`
			NodeType string `yaml:"node_type"`
			Quantity int    `yaml:"quantity"`
			VMSize   string `yaml:"vm_size"`
			Subnet   string `yaml:"subnet"`
			SSHKey   string `yaml:"ssh_key"`
			Spot     bool   `yaml:"spot"`
		} `yaml:"kube_node"`
	} `yaml:"nodes"`
}

// Bastion represents the bastion VM
type Bastion struct {
	AmiID             string   `yaml:"ami_id"`
	VMSize            string   `yaml:"vm_size"`
	AllowedCIDRBlocks []string `yaml:"allowedCIDRBlocks"`
}

// CreateKEOSDescriptor creates the keos.yaml file
func CreateKEOSDescriptor() error {

	// Read cluster.yaml file

	descriptorRAW, err := os.ReadFile("./cluster.yaml")
	if err != nil {
		return err
	}

	var descriptorFile DescriptorFile
	err = yaml.Unmarshal(descriptorRAW, &descriptorFile)
	if err != nil {
		return err
	}

	// Process the external registry
	var externalRegistryEntry string
	if len(descriptorFile.ExternalRegistry) > 0 {
		externalRegistryRAW, _ := yaml.Marshal(descriptorFile.ExternalRegistry)
		re := regexp.MustCompile(`\r?\n|^`)
		externalRegistryData := re.ReplaceAllString(strings.TrimSuffix(string(externalRegistryRAW), "\n"), "\n  ")
		externalRegistryEntry = "\nexternal_registry:" + externalRegistryData
	}

	// Process the cluster ID
	var clusterIDEntry string
	if len(descriptorFile.ClusterID) > 0 {
		clusterIDEntry = "\n  cluster_id: " + descriptorFile.ClusterID
	}

	// Process the domain
	var domainEntry string
	if len(descriptorFile.Keos.Domain) > 0 {
		domainEntry = "\n  domain: " + descriptorFile.Keos.Domain
	}

	// Process the external domain
	var externalDomainEntry string
	if len(descriptorFile.Keos.ExternalDomain) > 0 {
		externalDomainEntry = "\n  external_domain: " + descriptorFile.Keos.ExternalDomain
	}

	// Process the flavour
	var flavourEntry string
	if len(descriptorFile.Keos.Flavour) > 0 {
		flavourEntry = "\n  flavour: " + descriptorFile.Keos.Flavour
	}

	keosYAMLData := `---
k8s_bootstrapping:
  type: external
aws:
  eks: true
  enabled: true` +
		externalRegistryEntry + `
keos:` +
		clusterIDEntry +
		flavourEntry +
		domainEntry +
		externalDomainEntry + `
  storage:
    default_storage_class: gp2
    providers:
    - custom
`

	// Write file to disk
	err = os.WriteFile("keos.yaml", []byte(keosYAMLData), 0644)
	if err != nil {
		fmt.Println("failed writing keos.yaml file")
	}

	return nil
}
