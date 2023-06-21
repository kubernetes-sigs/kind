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

package createworker

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/commons"
)

type KEOSDescriptor struct {
	ExternalRegistry struct {
		AuthRequired bool   `yaml:"auth_required"`
		Type         string `yaml:"type"`
		URL          string `yaml:"url"`
	} `yaml:"external_registry"`
	AWS struct {
		Enabled bool `yaml:"enabled"`
		EKS     bool `yaml:"eks"`
	} `yaml:"aws,omitempty"`
	Azure struct {
		Enabled       bool   `yaml:"enabled"`
		AKS           bool   `yaml:"aks"`
		ResourceGroup string `yaml:"resource_group"`
	} `yaml:"azure,omitempty"`
	GCP struct {
		Enabled bool `yaml:"enabled"`
		GKE     bool `yaml:"gke"`
	} `yaml:"gcp,omitempty"`
	Keos struct {
		Calico struct {
			Ipip                 bool   `yaml:"ipip,omitempty"`
			VXLan                bool   `yaml:"vxlan,omitempty"`
			Pool                 string `yaml:"pool,omitempty"`
			DeployTigeraOperator bool   `yaml:"deploy_tigera_operator"`
		} `yaml:"calico"`
		ClusterID string `yaml:"cluster_id"`
		Dns       struct {
			ExternalDns struct {
				Enabled *bool `yaml:"enabled,omitempty"`
			} `yaml:"external_dns,omitempty"`
		} `yaml:"dns,omitempty"`
		// PR fixing exclude_if behaviour https://github.com/go-playground/validator/pull/939
		Domain          string `yaml:"domain,omitempty"`
		ExternalDomain  string `yaml:"external_domain,omitempty"`
		Flavour         string `yaml:"flavour"`
		K8sInstallation bool   `yaml:"k8s_installation"`
		Storage         struct {
			DefaultStorageClass string   `yaml:"default_storage_class,omitempty"`
			Providers           []string `yaml:"providers"`
			Config              struct {
				CSIAWS struct {
					EFS      []EFSConfig `yaml:"efs"`
					KMSKeyID string      `yaml:"kms_key_id,omitempty"`
				} `yaml:"csi-aws"`
			} `yaml:"config,omitempty"`
		} `yaml:"storage"`
	}
}

type EFSConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Permissions string `yaml:"permissions"`
}

func createKEOSDescriptor(descriptorFile commons.DescriptorFile, storageClass string) error {

	var keosDescriptor KEOSDescriptor
	var err error

	// External registry
	for _, registry := range descriptorFile.DockerRegistries {
		if registry.KeosRegistry {
			keosDescriptor.ExternalRegistry.URL = registry.URL
			keosDescriptor.ExternalRegistry.AuthRequired = registry.AuthRequired
			keosDescriptor.ExternalRegistry.Type = registry.Type
		}
	}

	// AWS
	if descriptorFile.InfraProvider == "aws" {
		keosDescriptor.AWS.Enabled = true
		keosDescriptor.AWS.EKS = descriptorFile.ControlPlane.Managed
	}

	// Azure
	if descriptorFile.InfraProvider == "azure" {
		keosDescriptor.Azure.Enabled = true
		keosDescriptor.Azure.AKS = descriptorFile.ControlPlane.Managed
		keosDescriptor.Azure.ResourceGroup = descriptorFile.ClusterID
	}

	// GCP
	if descriptorFile.InfraProvider == "gcp" {
		keosDescriptor.GCP.Enabled = true
		keosDescriptor.GCP.GKE = descriptorFile.ControlPlane.Managed
	}

	// Keos
	keosDescriptor.Keos.ClusterID = descriptorFile.ClusterID
	keosDescriptor.Keos.Domain = "cluster.local"
	if descriptorFile.ExternalDomain != "" {
		keosDescriptor.Keos.ExternalDomain = descriptorFile.ExternalDomain
	}
	keosDescriptor.Keos.Flavour = descriptorFile.Keos.Flavour

	// Keos - Calico
	if !descriptorFile.ControlPlane.Managed {
		if descriptorFile.InfraProvider == "azure" {
			keosDescriptor.Keos.Calico.VXLan = true
		} else {
			keosDescriptor.Keos.Calico.Ipip = true
		}
		if descriptorFile.Networks.PodsCidrBlock != "" {
			keosDescriptor.Keos.Calico.Pool = descriptorFile.Networks.PodsCidrBlock
		} else {
			keosDescriptor.Keos.Calico.Pool = "192.168.0.0/16"
		}
	}
	keosDescriptor.Keos.Calico.DeployTigeraOperator = false

	// Keos - Storage
	keosDescriptor.Keos.Storage.DefaultStorageClass = storageClass
	if descriptorFile.StorageClass.EFS.Name != "" {
		keosDescriptor.Keos.Storage.Providers = []string{"csi-aws"}

		name := descriptorFile.StorageClass.EFS.Name
		id := descriptorFile.StorageClass.EFS.ID
		permissions := descriptorFile.StorageClass.EFS.Permissions

		if permissions == "" {
			permissions = "700"
		}
		keosDescriptor.Keos.Storage.Config.CSIAWS.EFS = []EFSConfig{
			{
				Name:        name,
				ID:          id,
				Permissions: permissions,
			},
		}
		if descriptorFile.StorageClass.EncryptionKey != "" {
			keosDescriptor.Keos.Storage.Config.CSIAWS.KMSKeyID = descriptorFile.StorageClass.EncryptionKey
		}
	} else {
		keosDescriptor.Keos.Storage.Providers = []string{"custom"}
	}

	// Keos - External dns
	if !descriptorFile.Dns.ManageZone {
		keosDescriptor.Keos.Dns.ExternalDns.Enabled = &descriptorFile.Dns.ManageZone
	}

	keosYAMLData, err := yaml.Marshal(keosDescriptor)
	if err != nil {
		return err
	}

	// Rotate keos.yaml
	keosFilename := "keos.yaml"

	if _, err := os.Stat(keosFilename); err == nil {
		timestamp := time.Now().Format("2006-01-02@15:04:05")
		backupKeosFilename := keosFilename + "." + timestamp + "~"
		originalKeosFilePath := filepath.Join(".", keosFilename)
		backupKeosFilePath := filepath.Join(".", backupKeosFilename)

		if err := os.Rename(originalKeosFilePath, backupKeosFilePath); err != nil {
			return err
		}
	}

	// Write file to disk
	err = os.WriteFile("keos.yaml", []byte(keosYAMLData), 0644)
	if err != nil {
		return err
	}

	return nil
}
