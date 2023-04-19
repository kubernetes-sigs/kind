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
	"reflect"
	"strings"
	"text/template"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var ctel embed.FS

type K8sObject struct {
	APIVersion string         `yaml:"apiVersion" validate:"required"`
	Kind       string         `yaml:"kind" validate:"required"`
	Spec       DescriptorFile `yaml:"spec" validate:"required,dive"`
}

// DescriptorFile represents the YAML structure in the spec field of the descriptor file
type DescriptorFile struct {
	ClusterID        string `yaml:"cluster_id" validate:"required,min=3,max=100"`
	DeployAutoscaler bool   `yaml:"deploy_autoscaler" validate:"boolean"`

	Bastion Bastion `yaml:"bastion"`

	Credentials Credentials `yaml:"credentials"`

	InfraProvider string `yaml:"infra_provider" validate:"required,oneof='aws' 'gcp' 'azure'"`

	K8SVersion string `yaml:"k8s_version" validate:"required,startswith=v,min=7,max=8"`
	Region     string `yaml:"region" validate:"required"`

	Networks Networks `yaml:"networks"`

	Dns struct {
		ManageZone bool `yaml:"manage_zone" validate:"boolean"`
	} `yaml:"dns"`

	DockerRegistries []DockerRegistry `yaml:"docker_registries" validate:"dive"`

	ExternalDomain string `yaml:"external_domain" validate:"omitempty,hostname"`

	Keos struct {
		// PR fixing exclude_if behaviour https://github.com/go-playground/validator/pull/939
		Domain  string `yaml:"domain" validate:"omitempty,hostname"`
		Flavour string `yaml:"flavour"`
		Version string `yaml:"version"`
	} `yaml:"keos"`

	ControlPlane struct {
		Managed         bool   `yaml:"managed" validate:"boolean"`
		Name            string `yaml:"name"`
		NodeImage       string `yaml:"node_image" validate:"required_if=InfraProvider gcp"`
		HighlyAvailable bool   `yaml:"highly_available" validate:"boolean"`
		Size            string `yaml:"size" validate:"required_if=Managed false"`
		RootVolume      struct {
			Size      int    `yaml:"size" validate:"numeric"`
			Type      string `yaml:"type"`
			Encrypted bool   `yaml:"encrypted" validate:"boolean"`
		} `yaml:"root_volume"`
		AWS          AWS           `yaml:"aws"`
		ExtraVolumes []ExtraVolume `yaml:"extra_volumes"`
	} `yaml:"control_plane"`

	WorkerNodes WorkerNodes `yaml:"worker_nodes" validate:"required,dive"`
}

type Networks struct {
	VPCID                      string            `yaml:"vpc_id"`
	CidrBlock                  string            `yaml:"cidr,omitempty"`
	Tags                       map[string]string `yaml:"tags,omitempty"`
	AvailabilityZoneUsageLimit int               `yaml:"az_usage_limit" validate:"numeric"`
	AvailabilityZoneSelection  string            `yaml:"az_selection" validate:"oneof='Ordered' 'Random' '' "`

	Subnets []Subnets `yaml:"subnets"`
}

type Subnets struct {
	SubnetId         string            `yaml:"subnet_id"`
	AvailabilityZone string            `yaml:"az,omitempty"`
	IsPublic         *bool             `yaml:"is_public,omitempty"`
	RouteTableId     string            `yaml:"route_table_id,omitempty"`
	NatGatewayId     string            `yaml:"nat_id,omitempty"`
	Tags             map[string]string `yaml:"tags,omitempty"`
	CidrBlock        string            `yaml:"cidr,omitempty"`
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
	Name             string            `yaml:"name" validate:"required"`
	NodeImage        string            `yaml:"node_image" validate:"required_if=InfraProvider gcp"`
	Quantity         int               `yaml:"quantity" validate:"required,numeric,gt=0"`
	Size             string            `yaml:"size" validate:"required"`
	ZoneDistribution string            `yaml:"zone_distribution" validate:"omitempty,oneof='balanced' 'unbalanced'"`
	AZ               string            `yaml:"az"`
	SSHKey           string            `yaml:"ssh_key"`
	Spot             bool              `yaml:"spot" validate:"omitempty,boolean"`
	Labels           map[string]string `yaml:"labels"`
	NodeGroupMaxSize int               `yaml:"max_size" validate:"omitempty,numeric,required_with=NodeGroupMinSize,gtefield=Quantity,gt=0"`
	NodeGroupMinSize int               `yaml:"min_size" validate:"omitempty,numeric,required_with=NodeGroupMaxSize,ltefield=Quantity,gt=0"`
	RootVolume       struct {
		Size      int    `yaml:"size" validate:"numeric"`
		Type      string `yaml:"type"`
		Encrypted bool   `yaml:"encrypted" validate:"boolean"`
	} `yaml:"root_volume"`
	ExtraVolumes []ExtraVolume `yaml:"extra_volumes"`
}

// Bastion represents the bastion VM
type Bastion struct {
	NodeImage         string   `yaml:"node_image"`
	VMSize            string   `yaml:"vm_size"`
	AllowedCIDRBlocks []string `yaml:"allowedCIDRBlocks"`
	SSHKey            string   `yaml:"ssh_key"`
}

type Node struct {
	AZ      string
	QA      int
	MaxSize int
	MinSize int
}

type ExtraVolume struct {
	DeviceName string `yaml:"device_name"`
	Size       int    `yaml:"size" validate:"numeric"`
	Type       string `yaml:"type"`
	Label      string `yaml:"label"`
	Encrypted  bool   `yaml:"encrypted" validate:"boolean"`
	MountPath  string `yaml:"mount_path" validate:"omitempty,required_with=Name"`
}

type Credentials struct {
	AWS              AWSCredentials              `yaml:"aws"`
	GCP              GCPCredentials              `yaml:"gcp"`
	GithubToken      string                      `yaml:"github_token"`
	DockerRegistries []DockerRegistryCredentials `yaml:"docker_registries"`
}

type AWSCredentials struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	Account   string `yaml:"account"`
}

type GCPCredentials struct {
	ProjectID    string `yaml:"project_id"`
	PrivateKeyID string `yaml:"private_key_id"`
	PrivateKey   string `yaml:"private_key"`
	ClientEmail  string `yaml:"client_email"`
	ClientID     string `yaml:"client_id"`
}

type DockerRegistryCredentials struct {
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type DockerRegistry struct {
	AuthRequired bool   `yaml:"auth_required" validate:"boolean"`
	Type         string `yaml:"type"`
	URL          string `yaml:"url" validate:"required"`
	KeosRegistry bool   `yaml:"keos_registry" validate:"boolean"`
}

type TemplateParams struct {
	Descriptor       DescriptorFile
	Credentials      map[string]string
	DockerRegistries []map[string]interface{}
}

// Init sets default values for the DescriptorFile
func (d DescriptorFile) Init() DescriptorFile {
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

	// Managed zones
	d.Dns.ManageZone = true

	return d
}

// Read descriptor file
func GetClusterDescriptor(descriptorPath string) (*DescriptorFile, error) {

	var k8sStruct K8sObject

	descriptorRAW, err := os.ReadFile(descriptorPath)
	if err != nil {
		return nil, err
	}

	k8sStruct.Spec = new(DescriptorFile).Init()
	err = yaml.Unmarshal(descriptorRAW, &k8sStruct)
	if err != nil {
		return nil, err
	}
	descriptorFile := k8sStruct.Spec

	validate := validator.New()
	err = validate.Struct(k8sStruct)
	if err != nil {
		return nil, err
	}

	return &descriptorFile, nil
}

func resto(n int, i int, azs int) int {
	var r int
	r = (n % azs) / (i + 1)
	if r > 1 {
		r = 1
	}
	return r
}

func GetClusterManifest(flavor string, params TemplateParams, azs []string) (string, error) {
	funcMap := template.FuncMap{
		"loop": func(az string, zd string, qa int, maxsize int, minsize int) <-chan Node {
			ch := make(chan Node)
			go func() {
				var q int
				var mx int
				var mn int
				if az != "" {
					ch <- Node{AZ: az, QA: qa, MaxSize: maxsize, MinSize: minsize}
				} else {
					for i, a := range azs {
						if zd == "unbalanced" {
							q = qa/len(azs) + resto(qa, i, len(azs))
							mx = maxsize/len(azs) + resto(maxsize, i, len(azs))
							mn = minsize/len(azs) + resto(minsize, i, len(azs))
							ch <- Node{AZ: a, QA: q, MaxSize: mx, MinSize: mn}
						} else {
							ch <- Node{AZ: a, QA: qa / len(azs), MaxSize: maxsize / len(azs), MinSize: minsize / len(azs)}
						}
					}
				}
				close(ch)
			}()
			return ch
		},
		"hostname": func(s string) string {
			return strings.Split(s, "/")[0]
		},
		"checkReference": func(v interface{}) bool {
			defer func() { recover() }()
			return v != nil && !reflect.ValueOf(v).IsNil() && v != "nil" && v != "<nil>"
		},
		"isNotEmpty": func(v interface{}) bool {
			return !reflect.ValueOf(v).IsZero()
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
