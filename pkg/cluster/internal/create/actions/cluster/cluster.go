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
	"os"
	"text/template"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var ctel embed.FS

// DescriptorFile represents the YAML structure in the descriptor file
type DescriptorFile struct {
	APIVersion       string `yaml:"apiVersion"`
	Kind             string `yaml:"kind"`
	ClusterID        string `yaml:"cluster_id" validate:"required,min=3,max=100"`
	DeployAutoscaler bool   `yaml:"deploy_autoscaler" validate:"boolean"`

	Bastion Bastion `yaml:"bastion"`

	Credentials Credentials `yaml:"credentials"`
	GithubToken string      `yaml:"github_token"`

	InfraProvider string `yaml:"infra_provider" validate:"required,oneof='aws' 'gcp' 'azure'"`

	K8SVersion   string `yaml:"k8s_version" validate:"required,startswith=v,min=7,max=8"`
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

	ExternalRegistry ExternalRegistry `yaml:"external_registry" validate:"dive"`

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
		RootVolume      struct {
			Size      int    `yaml:"size" validate:"numeric"`
			Type      string `yaml:"type"`
			Encrypted bool   `yaml:"encrypted" validate:"boolean"`
		} `yaml:"root_volume"`
		AWS AWS `yaml:"aws"`
	} `yaml:"control_plane"`

	WorkerNodes WorkerNodes `yaml:"worker_nodes" validate:"required,dive"`
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
	Quantity         int    `yaml:"quantity" validate:"required,numeric,gt=0"`
	Size             string `yaml:"size" validate:"required"`
	Image            string `yaml:"image" validate:"required_if=InfraProvider gcp"`
	ZoneDistribution string `yaml:"zone_distribution" validate:"omitempty,oneof='balanced' 'unbalanced'"`
	AZ               string `yaml:"az"`
	SSHKey           string `yaml:"ssh_key"`
	Spot             bool   `yaml:"spot" validate:"omitempty,boolean"`
	NodeGroupMaxSize int    `yaml:"max_size" validate:"required,numeric"`
	NodeGroupMinSize int    `yaml:"min_size" validate:"required,numeric"`
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
	AZ      string
	QA      int
	MaxSize int
	MinSize int
}

type Credentials struct {
	// AWS
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	Account   string `yaml:"account"`

	// GCP
	ProjectID    string `yaml:"project_id"`
	PrivateKeyID string `yaml:"private_key_id"`
	PrivateKey   string `yaml:"private_key"`
	ClientEmail  string `yaml:"client_email"`
	ClientID     string `yaml:"client_id"`
}

type ExternalRegistry struct {
	AuthRequired bool   `yaml:"auth_required" validate:"boolean"`
	Type         string `yaml:"type"`
	URL          string `yaml:"url" validate:"required"`
	User         string `yaml:"user"`
	Pass         string `yaml:"pass"`
}

type TemplateParams struct {
	Descriptor       DescriptorFile
	Credentials      map[string]string
	ExternalRegistry map[string]string
}

// Init sets default values for the DescriptorFile
func (d DescriptorFile) Init() DescriptorFile {
	d.FullyPrivate = false
	d.ControlPlane.HighlyAvailable = true

	// Autoscaler
	d.DeployAutoscaler = true

	// EKS
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

func GetClusterManifest(flavor string, params TemplateParams) (string, error) {

	funcMap := template.FuncMap{
		"loop": func(az string, qa int, maxsize int, minsize int) <-chan Node {
			ch := make(chan Node)
			go func() {
				var azs []string
				var q int
				var mx int
				var mn int
				if az != "" {
					azs = []string{az}
					q = qa
					mx = maxsize
					mn = minsize
				} else {
					azs = []string{"a", "b", "c"}
					q = qa / 3
					mx = maxsize / 3
					mn = minsize / 3
				}
				for _, a := range azs {
					ch <- Node{AZ: a, QA: q, MaxSize: mx, MinSize: mn}
				}
				close(ch)
			}()
			return ch
		},
	}

	var tpl bytes.Buffer
	t, err := template.New("").Funcs(funcMap).ParseFS(ctel, "templates/"+flavor)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, flavor, params)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}
