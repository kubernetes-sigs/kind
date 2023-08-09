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
	DockerRegistry struct {
		AuthRequired bool   `yaml:"auth_required"`
		Type         string `yaml:"type"`
		URL          string `yaml:"url"`
	} `yaml:"docker_registry"`
	HelmRepository struct {
		AuthRequired bool   `yaml:"auth_required"`
		URL          string `yaml:"url"`
	} `yaml:"helm_repository"`
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

func createKEOSDescriptor(keosCluster commons.KeosCluster, storageClass string) error {

	var keosDescriptor KEOSDescriptor
	var err error

	// External registry
	for _, registry := range keosCluster.Spec.DockerRegistries {
		if registry.KeosRegistry {
			keosDescriptor.DockerRegistry.URL = registry.URL
			keosDescriptor.DockerRegistry.AuthRequired = registry.AuthRequired
			keosDescriptor.DockerRegistry.Type = registry.Type
		}
	}

	// Helm repository
	keosDescriptor.HelmRepository.URL = keosCluster.Spec.HelmRepository.URL
	keosDescriptor.HelmRepository.AuthRequired = keosCluster.Spec.HelmRepository.AuthRequired

	// AWS
	if keosCluster.Spec.InfraProvider == "aws" {
		keosDescriptor.AWS.Enabled = true
		keosDescriptor.AWS.EKS = keosCluster.Spec.ControlPlane.Managed
	}

	// Azure
	if keosCluster.Spec.InfraProvider == "azure" {
		keosDescriptor.Azure.Enabled = true
		keosDescriptor.Azure.AKS = keosCluster.Spec.ControlPlane.Managed
		keosDescriptor.Azure.ResourceGroup = keosCluster.Metadata.Name
	}

	// GCP
	if keosCluster.Spec.InfraProvider == "gcp" {
		keosDescriptor.GCP.Enabled = true
		keosDescriptor.GCP.GKE = keosCluster.Spec.ControlPlane.Managed
	}

	// Keos
	keosDescriptor.Keos.ClusterID = keosCluster.Metadata.Name
	keosDescriptor.Keos.Domain = "cluster.local"
	if keosCluster.Spec.ExternalDomain != "" {
		keosDescriptor.Keos.ExternalDomain = keosCluster.Spec.ExternalDomain
	}
	keosDescriptor.Keos.Flavour = keosCluster.Spec.Keos.Flavour

	// Keos - Calico
	if !keosCluster.Spec.ControlPlane.Managed {
		if keosCluster.Spec.InfraProvider == "azure" {
			keosDescriptor.Keos.Calico.VXLan = true
		} else {
			keosDescriptor.Keos.Calico.Ipip = true
		}
		if keosCluster.Spec.Networks.PodsCidrBlock != "" {
			keosDescriptor.Keos.Calico.Pool = keosCluster.Spec.Networks.PodsCidrBlock
		} else {
			keosDescriptor.Keos.Calico.Pool = "192.168.0.0/16"
		}
	}
	keosDescriptor.Keos.Calico.DeployTigeraOperator = false

	// Keos - Storage
	keosDescriptor.Keos.Storage.DefaultStorageClass = storageClass
	if keosCluster.Spec.StorageClass.EFS.Name != "" {
		keosDescriptor.Keos.Storage.Providers = []string{"csi-aws"}

		name := keosCluster.Spec.StorageClass.EFS.Name
		id := keosCluster.Spec.StorageClass.EFS.ID
		permissions := keosCluster.Spec.StorageClass.EFS.Permissions

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
		if keosCluster.Spec.StorageClass.EncryptionKey != "" {
			keosDescriptor.Keos.Storage.Config.CSIAWS.KMSKeyID = keosCluster.Spec.StorageClass.EncryptionKey
		}
	} else {
		keosDescriptor.Keos.Storage.Providers = []string{"custom"}
	}

	// Keos - External dns
	if !keosCluster.Spec.Dns.ManageZone {
		keosDescriptor.Keos.Dns.ExternalDns.Enabled = &keosCluster.Spec.Dns.ManageZone
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
