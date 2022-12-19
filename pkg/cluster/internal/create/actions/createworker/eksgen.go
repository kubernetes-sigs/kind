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

// Package createworker implements the create worker action
package createworker

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	goyaml "github.com/go-yaml/yaml"
	"sigs.k8s.io/kind/pkg/errors"
)

// K8sObject represents the Kubernetes manifest
type K8sObject struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec map[string]interface{} `yaml:"spec"`
}

// MachineDeployment represents the MachineDeployment manifest
type MachineDeployment struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec MachineDeploymentSpec `yaml:"spec"`
}

// MachineDeploymentSpec represents the spec of a MachineDeployment manifest
type MachineDeploymentSpec struct {
	ClusterName string `yaml:"clusterName"`
	Replicas    int    `yaml:"replicas"`
	Selector    struct {
		MatchLabels interface{} `yaml:"matchLabels"`
	} `yaml:"selector"`
	Template struct {
		Spec struct {
			Bootstrap struct {
				ConfigRef struct {
					APIVersion string `yaml:"apiVersion"`
					Kind       string `yaml:"kind"`
					Name       string `yaml:"name"`
					Namespace  string `yaml:"namespace"`
				} `yaml:"configRef"`
			} `yaml:"bootstrap"`
			ClusterName       string `yaml:"clusterName"`
			InfrastructureRef struct {
				APIVersion string `yaml:"apiVersion"`
				Kind       string `yaml:"kind"`
				Name       string `yaml:"name"`
				Namespace  string `yaml:"namespace"`
			} `yaml:"infrastructureRef"`
			Version       string `yaml:"version"`
			FailureDomain string `yaml:"failureDomain"`
		} `yaml:"spec"`
	} `yaml:"template"`
}

// downloadTemplates downloads the template manifests according to the CAPA version provided
func downloadTemplates(capaVersion string) ([]byte, error) {
	// url := "https://raw.githubusercontent.com/kubernetes-sigs/cluster-api-provider-aws/" + capaVersion + "/templates/cluster-template-eks.yaml"
	url := "https://raw.githubusercontent.com/kubernetes-sigs/cluster-api-provider-aws/" + capaVersion + "/templates/cluster-template-eks.yaml"

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		// return nil, fmt.Errorf("bad status: %s", resp.Status)
		return nil, errors.Wrap(err, "failed to get templates - bad status: "+resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// generateEKSManifest downloads, renders and persists the manifests for the EKS cluster
func generateEKSManifest(secretsFile SecretsFile, descriptorFile DescriptorFile, capiClustersNamespace string) (string, error) {

	// TODO: Embeber los templates?
	// TODO: Obtener capaVersion del cluster?
	capaVersion := "v1.5.1"
	templates, _ := downloadTemplates(capaVersion)
	templatesRAW := goyaml.NewDecoder(bytes.NewReader(templates))

	nodesRegion := secretsFile.Secrets.AWS.Credentials.Region
	// TODO: Get zones from the AWS endpoint
	nodesZones := []string{nodesRegion + "a", nodesRegion + "b", nodesRegion + "c"}

	// Process the YAML file with multiple manifests
	var eksDescriptorData string
	for {
		// Decode the next manifest until the EOF or error
		var manifestRAW interface{}
		err := templatesRAW.Decode(&manifestRAW)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Marshall the template manifest (for rendering)
		manifestBytes, err := goyaml.Marshal(manifestRAW)
		if err != nil {
			return "", err
		}
		manifestStr := string(manifestBytes)

		// Render env vars (this needs a string manifest)
		var envMap = map[string]string{
			"CLUSTER_NAME":          descriptorFile.ClusterID,
			"AWS_REGION":            nodesRegion,
			"AWS_SSH_KEY_NAME":      descriptorFile.Nodes.KubeNode.SSHKey,
			"KUBERNETES_VERSION":    descriptorFile.K8SVersion,
			"WORKER_MACHINE_COUNT":  strconv.Itoa(descriptorFile.Nodes.KubeNode.Quantity),
			"AWS_NODE_MACHINE_TYPE": descriptorFile.Nodes.KubeNode.VMSize,
		}

		if descriptorFile.Nodes.KubeNode.SSHKey == "" {
			envMap["AWS_SSH_KEY_NAME"] = "\\\"\\\""
		}

		for k, e := range envMap {
			manifestStr = strings.Replace(manifestStr, "${"+k+"}", e, -1)
		}
		renderedStr := manifestStr

		// Unmarshal rendered manifest for filtering
		var k8sObject K8sObject
		err = goyaml.Unmarshal([]byte(renderedStr), &k8sObject)
		if err != nil {
			// fmt.Printf("nope %v", err)
			return "", err
		}

		// Process AWSManagedControlPlane
		if k8sObject.Kind == "AWSManagedControlPlane" {
			// Add specs in AWSManagedControlPlane object (needs marshaling)
			specBytes, err := goyaml.Marshal(k8sObject.Spec)
			if err != nil {
				// fmt.Printf("nope %v", err)
				return "", err
			}
			// fmt.Printf("STG %s", string(specBytes))
			specBytes = append(specBytes, []byte("associateOIDCProvider: yes")...)
			specBytes = append(specBytes, []byte("\naddons:\n  - name: aws-ebs-csi-driver\n    version: v1.11.4-eksbuild.1")...)
			// fmt.Printf("\nSTG %v", descriptorFile.Bastion)

			// Note: we take the VMSize as a mandatory parameter
			if descriptorFile.Bastion.VMSize != "" {
				// fmt.Printf("\nSTG: BASTION.")
				specBytes = append(specBytes, []byte("\nbastion:\n  enabled: true"+
					"\n  instanceType: "+descriptorFile.Bastion.VMSize+
					"\n  ami: "+descriptorFile.Bastion.AmiID)...)
				if descriptorFile.Bastion.AllowedCIDRBlocks != nil {
					specBytes = append(specBytes, []byte("\n  allowedCIDRBlocks: \n  - "+strings.Join(descriptorFile.Bastion.AllowedCIDRBlocks, "\n  - "))...)
				}
			}

			networkBytes := "\nnetwork:\n  vpc:\n    availabilityZoneSelection: Ordered\n    availabilityZoneUsageLimit: 3"
			// Add custom VPC
			if descriptorFile.Networks.VPCID != "" {
				networkBytes += "\n    id: " + descriptorFile.Networks.VPCID
			}
			specBytes = append(specBytes, []byte(networkBytes)...)

			var specMap map[string]interface{}
			err = goyaml.Unmarshal(specBytes, &specMap)
			k8sObject.Spec = specMap
		}

		// Process MachineDeployment
		if k8sObject.Kind == "MachineDeployment" {

			// Unmarshal rendered manifest for filtering
			var machineDeployment MachineDeployment
			err = goyaml.Unmarshal([]byte(renderedStr), &machineDeployment)
			if err != nil {
				// fmt.Printf("nope %v", err)
				return "", err
			}

			// Split the MachineDeployment object into zones
			for i, nodeZone := range nodesZones {
				machineDeploymentZone := machineDeployment
				machineDeploymentName := descriptorFile.ClusterID + "-md-" + strconv.Itoa(i)

				machineDeploymentZone.Metadata.Name = machineDeploymentName
				machineDeploymentZone.Spec.Replicas = machineDeployment.Spec.Replicas / 3
				machineDeploymentZone.Spec.Template.Spec.Bootstrap.ConfigRef.Name = machineDeploymentName
				machineDeploymentZone.Spec.Template.Spec.Bootstrap.ConfigRef.Namespace = capiClustersNamespace
				machineDeploymentZone.Spec.Template.Spec.InfrastructureRef.Name = machineDeploymentName
				machineDeploymentZone.Spec.Template.Spec.InfrastructureRef.Namespace = capiClustersNamespace
				machineDeploymentZone.Spec.Template.Spec.FailureDomain = nodeZone
				// machineDeploymentZone.Spec.Selector.MatchLabels = "null"

				// // Add specs in MachineDeployment object (needs marshaling)
				// specBytesZone, err := goyaml.Marshal(machineDeploymentZone.Spec)
				// if err != nil {
				// 	// fmt.Printf("nope %v", err)
				// 	return "", err
				// }
				// specBytesZone = append(specBytesZone, []byte("selector:\n  matchLabels:")...)

				// var specMapZone MachineDeploymentSpec
				// err = goyaml.Unmarshal(specBytesZone, &specMapZone)
				// machineDeploymentZone.Spec = specMapZone

				machineDeploymentZoneBytes, _ := goyaml.Marshal(machineDeploymentZone)
				eksDescriptorData += "---\n" + string(machineDeploymentZoneBytes)
			}
		}

		// Process AWSMachineTemplate or EKSConfigTemplate
		if k8sObject.Kind == "AWSMachineTemplate" || k8sObject.Kind == "EKSConfigTemplate" {
			for i := range nodesZones {
				k8sObjectZone := k8sObject

				// TODO: Add ssh option

				// TODO: Add spot option
				//  awsMachineTemplate
				// spec:
				//   template:
				//     spec:
				// 	    iamInstanceProfile: nodes.cluster-api-provider-aws.sigs.k8s.io
				// 	    instanceType: t3.medium
				// 	    spotMarketOptions:
				// 	      maxPrice: ""
				// 	    sshKeyName: stg-capi

				k8sObjectZone.Metadata.Name = descriptorFile.ClusterID + "-md-" + strconv.Itoa(i)

				k8sObjectZoneBytes, _ := goyaml.Marshal(k8sObjectZone)
				eksDescriptorData += "---\n" + string(k8sObjectZoneBytes)
			}
		}

		if k8sObject.Kind != "MachineDeployment" && k8sObject.Kind != "AWSMachineTemplate" && k8sObject.Kind != "EKSConfigTemplate" {
			k8sObjectBytes, _ := goyaml.Marshal(k8sObject)
			eksDescriptorData += "---\n" + string(k8sObjectBytes)
		}
	}
	return eksDescriptorData, nil
}
