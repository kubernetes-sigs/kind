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

package cluster

import (
	"bytes"
	"embed"
	"errors"
	"os"
	"strings"
	"text/template"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var ctel embed.FS

// DescriptorFile represents the YAML structure in the descriptor file
type DescriptorFile struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	ClusterID  string `yaml:"cluster_id" validate:"required,min=3,max=100"`

	Bastion Bastion `yaml:"bastion"`

	AWSCredentials AWSCredentials `yaml:"aws"`
	GithubToken    string         `yaml:"github_token"`

	InfraProvider string `yaml:"infra_provider" validate:"required,oneof='aws' 'gcp' 'azure'"`

	K8SVersion   string `yaml:"k8s_version" validate:"required,startswith=v,len=7"`
	Region       string `yaml:"region" validate:"required"`
	SSHKey       string `yaml:"ssh_key"`
	FullyPrivate bool   `yaml:"fully_private" validate:"boolean"`

	Networks struct {
		VPCID   string `yaml:"vpc_id" validate:"required_with=Subnets"`
		Subnets []struct {
			AvailabilityZone string `yaml:"availability_zone"`
			Name             string `yaml:"name"`
			PrivateCIDR      string `yaml:"private_cidr"`
			PublicCIDR       string `yaml:"public_cidr"`
		} `yaml:"subnets"`
	} `yaml:"networks"`

	ExternalRegistry struct {
		AuthRequired bool   `yaml:"auth_required" validate:"boolean"`
		Type         string `yaml:"type"`
		URL          string `yaml:"url" validate:"required"`
	} `yaml:"external_registry"`

	Keos struct {
		Domain         string `yaml:"domain" validate:"required,hostname"`
		ExternalDomain string `yaml:"external_domain" validate:"required,hostname"`
		Flavour        string `yaml:"flavour"`
		Version        string `yaml:"version"`
	} `yaml:"keos"`

	ControlPlane struct {
		Managed         bool   `yaml:"managed" validate:"boolean"`
		Name            string `yaml:"name"`
		AmiID           string `yaml:"ami_id"`
		HighlyAvailable bool   `yaml:"highly_available" validate:"boolean"`
		Size            string `yaml:"size" validate:"required_if=Managed false"`
		Image           string `yaml:"image" validate:"required_if=InfraProvider gcp"`
		AWS             AWS    `yaml:"aws"`
	} `yaml:"control_plane"`

	WorkerNodes WorkerNodes `yaml:"worker_nodes"`
}

type AWS struct {
	AssociateOIDCProvider bool `yaml:"associate_oidc_provider" validate:"boolean"`
	Logging               struct {
		ApiServer         bool `yaml:"api_server" validate:"boolean"`
		Audit             bool `yaml:"audit" validate:"boolean"`
		Authenticator     bool `yaml:"authenticator" validate:"boolean"`
		ControllerManager bool `yaml:"controller_manager" validate:"boolean"`
		Scheduler         bool `yaml:"scheduler" validate:"boolean"`
	} `yaml:"logging"`
}

type WorkerNodes []struct {
	Name             string `yaml:"name" validate:"required"`
	AmiID            string `yaml:"ami_id"`
	Quantity         int    `yaml:"quantity" validate:"required,numeric"`
	Size             string `yaml:"size" validate:"required"`
	Image            string `yaml:"image" validate:"required_if=InfraProvider gcp"`
	ZoneDistribution string `yaml:"zone_distribution" validate:"oneof='balanced' 'unbalanced'"`
	AZ               string `yaml:"az"`
	SSHKey           string `yaml:"ssh_key"`
	Spot             bool   `yaml:"spot" validate:"boolean"`
	RootVolume       struct {
		Size      int    `yaml:"size" validate:"numeric"`
		Type      string `yaml:"type"`
		Encrypted bool   `yaml:"encrypted" validate:"boolean"`
	} `yaml:"root_volume"`
}

// Bastion represents the bastion VM
type Bastion struct {
	AmiID             string   `yaml:"ami_id"`
	VMSize            string   `yaml:"vm_size"`
	AllowedCIDRBlocks []string `yaml:"allowedCIDRBlocks"`
}

type Node struct {
	AZ string
	QA int
}
type AWSCredentials struct {
	Credentials struct {
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		Region    string `yaml:"region"`
		AccountID string `yaml:"account_id"`
	} `yaml:"credentials"`
}

// Init sets default values for the DescriptorFile
func (d DescriptorFile) Init() DescriptorFile {
	d.FullyPrivate = false
	d.ControlPlane.HighlyAvailable = true

	// AWS
	d.ControlPlane.AWS.AssociateOIDCProvider = true
	d.ControlPlane.AWS.Logging.ApiServer = false
	d.ControlPlane.AWS.Logging.Audit = false
	d.ControlPlane.AWS.Logging.Authenticator = false
	d.ControlPlane.AWS.Logging.ControllerManager = false
	d.ControlPlane.AWS.Logging.Scheduler = false
	return d
}

// Read descriptor file
func GetClusterDescriptor(descriptorName string) (*DescriptorFile, error) {
	descriptorRAW, err := os.ReadFile("./" + descriptorName)
	if err != nil {
		return nil, err
	}
	descriptorFile := new(DescriptorFile).Init()
	err = yaml.Unmarshal(descriptorRAW, &descriptorFile)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	err = validate.Struct(descriptorFile)
	if err != nil {
		return nil, err
	}
	return &descriptorFile, nil
}

func getTemplateFile(d DescriptorFile) (string, error) {
	var t string
	switch d.InfraProvider {
	case "aws":
		if d.ControlPlane.Managed {
			t = "templates/aws.eks.tmpl"
		} else {
			return "", errors.New("AWS not supported yet")
		}
	case "gcp":
		return "", errors.New("GCP not supported yet")
	}
	return t, nil
}

func GetClusterManifest(d DescriptorFile) (string, error) {

	funcMap := template.FuncMap{
		"loop": func(az string, qa int) <-chan Node {
			ch := make(chan Node)
			go func() {
				var azs []string
				var q int
				if az != "" {
					azs = []string{az}
					q = qa
				} else {
					azs = []string{"a", "b", "c"}
					q = qa / 3
				}
				for _, a := range azs {
					ch <- Node{AZ: a, QA: q}
				}
				close(ch)
			}()
			return ch
		},
	}

	flavor, err := getTemplateFile(d)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	t, err := template.New("").Funcs(funcMap).ParseFS(ctel, flavor)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, strings.Split(flavor, "/")[1], d)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}
