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

package commons

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	vault "github.com/sosedoff/ansible-vault-go"
	"gopkg.in/yaml.v3"
)

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

	Credentials Credentials `yaml:"credentials" validate:"dive"`

	InfraProvider string `yaml:"infra_provider" validate:"required,oneof='aws' 'gcp' 'azure'"`

	K8SVersion string `yaml:"k8s_version" validate:"required,startswith=v,min=7,max=8"`
	Region     string `yaml:"region" validate:"required"`

	Networks struct {
		VPCID         string `yaml:"vpc_id"`
		PodsCidrBlock string `yaml:"pods_cidr" validate:"omitempty,cidrv4"`
		PodsSubnets   []struct {
			SubnetID string `yaml:"subnet_id"`
		} `yaml:"pods_subnets"`
		Subnets []struct {
			SubnetID string `yaml:"subnet_id"`
		} `yaml:"subnets"`
	} `yaml:"networks" validate:"omitempty,dive"`

	Dns struct {
		ManageZone bool `yaml:"manage_zone" validate:"boolean"`
	} `yaml:"dns"`

	DockerRegistries []DockerRegistry `yaml:"docker_registries" validate:"dive"`

	ExternalDomain string `yaml:"external_domain" validate:"omitempty,hostname"`

	Keos struct {
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
		AWS          AWSCP         `yaml:"aws"`
		Azure        AzureCP       `yaml:"azure"`
		ExtraVolumes []ExtraVolume `yaml:"extra_volumes"`
	} `yaml:"control_plane"`

	WorkerNodes WorkerNodes `yaml:"worker_nodes" validate:"required,dive"`
}

type AWSCP struct {
	AssociateOIDCProvider bool `yaml:"associate_oidc_provider" validate:"boolean"`
	Logging               struct {
		ApiServer         bool `yaml:"api_server" validate:"boolean"`
		Audit             bool `yaml:"audit" validate:"boolean"`
		Authenticator     bool `yaml:"authenticator" validate:"boolean"`
		ControllerManager bool `yaml:"controller_manager" validate:"boolean"`
		Scheduler         bool `yaml:"scheduler" validate:"boolean"`
	} `yaml:"logging"`
}

type AzureCP struct {
	IdentityID string `yaml:"identity_id"`
	Tier       string `yaml:"tier" validate:"oneof='Free' 'Paid'"`
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
	Taints           []string          `yaml:"taints" validate:"omitempty,dive"`
	NodeGroupMaxSize int               `yaml:"max_size" validate:"required_with=NodeGroupMinSize,numeric,omitempty"`
	NodeGroupMinSize int               `yaml:"min_size" validate:"required_with=NodeGroupMaxSize,numeric,omitempty"`
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

type ExtraVolume struct {
	Name       string `yaml:"name"`
	DeviceName string `yaml:"device_name"`
	Size       int    `yaml:"size" validate:"numeric"`
	Type       string `yaml:"type"`
	Label      string `yaml:"label"`
	Encrypted  bool   `yaml:"encrypted" validate:"boolean"`
	MountPath  string `yaml:"mount_path" validate:"omitempty,required_with=Name"`
}

type Credentials struct {
	AWS              AWSCredentials              `yaml:"aws" validate:"excluded_with=AZURE GCP"`
	AZURE            AzureCredentials            `yaml:"azure" validate:"excluded_with=AWS GCP"`
	GCP              GCPCredentials              `yaml:"gcp" validate:"excluded_with=AWS AZURE"`
	GithubToken      string                      `yaml:"github_token"`
	DockerRegistries []DockerRegistryCredentials `yaml:"docker_registries"`
}

type AWSCredentials struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	AccountID string `yaml:"account_id"`
}

type AzureCredentials struct {
	SubscriptionID string `yaml:"subscription_id"`
	TenantID       string `yaml:"tenant_id"`
	ClientID       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
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
	Type         string `yaml:"type" validate:"required,oneof='acr' 'ecr' 'generic'"`
	URL          string `yaml:"url" validate:"required"`
	KeosRegistry bool   `yaml:"keos_registry" validate:"omitempty,boolean"`
}

type TemplateParams struct {
	Descriptor       DescriptorFile
	Credentials      map[string]string
	DockerRegistries []map[string]interface{}
}

type AWS struct {
	Credentials AWSCredentials `yaml:"credentials"`
}

type AZURE struct {
	Credentials AzureCredentials `yaml:"credentials"`
}

type GCP struct {
	Credentials GCPCredentials `yaml:"credentials"`
}

type SecretsFile struct {
	Secrets Secrets `yaml:"secrets"`
}

type Secrets struct {
	AWS              AWS                         `yaml:"aws"`
	AZURE            AZURE                       `yaml:"azure"`
	GCP              GCP                         `yaml:"gcp"`
	GithubToken      string                      `yaml:"github_token"`
	ExternalRegistry DockerRegistryCredentials   `yaml:"external_registry"`
	DockerRegistries []DockerRegistryCredentials `yaml:"docker_registries"`
}

type ProviderParams struct {
	Region      string
	Managed     bool
	Credentials map[string]string
	GithubToken string
}

// Init sets default values for the DescriptorFile
func (d DescriptorFile) Init() DescriptorFile {
	d.ControlPlane.HighlyAvailable = true

	// AKS
	d.ControlPlane.Azure.Tier = "Paid"

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
	_, err := os.Stat(descriptorPath)
	if err != nil {
		return nil, errors.New("No exists any cluster descriptor as " + descriptorPath)
	}
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

	validate.RegisterCustomTypeFunc(CustomTypeAWSCredsFunc, AWSCredentials{})
	validate.RegisterCustomTypeFunc(CustomTypeGCPCredsFunc, GCPCredentials{})
	validate.RegisterValidation("gte_param_if_exists", gteParamIfExists)
	validate.RegisterValidation("lte_param_if_exists", lteParamIfExists)
	validate.RegisterValidation("required_if_for_bool", requiredIfForBool)

	err = validate.Struct(descriptorFile)
	if err != nil {
		return nil, err
	}
	return &descriptorFile, nil
}

func DecryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)

	if err != nil {
		return "", err
	}
	return data, nil
}

func GetSecretsFile(secretsPath string, vaultPassword string) (*SecretsFile, error) {
	secretRaw, err := DecryptFile(secretsPath, vaultPassword)
	var secretFile SecretsFile
	if err != nil {
		err := errors.New("the vaultPassword is incorrect")
		return nil, err
	}

	err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
	if err != nil {
		return nil, err
	}
	return &secretFile, nil
}

func IfExistsStructField(fl validator.FieldLevel) bool {
	structValue := reflect.ValueOf(fl.Parent().Interface())

	excludeFieldName := fl.Param()

	// Get the value of the exclude field
	excludeField := structValue.FieldByName(excludeFieldName)

	// Exclude field is set to false or invalid, so don't exclude this field
	return reflect.DeepEqual(excludeField, reflect.Zero(reflect.TypeOf(excludeField)).Interface())
}

func CustomTypeAWSCredsFunc(field reflect.Value) interface{} {
	if value, ok := field.Interface().(AWSCredentials); ok {
		return value.AccessKey
	}
	return nil
}

func CustomTypeGCPCredsFunc(field reflect.Value) interface{} {
	if value, ok := field.Interface().(GCPCredentials); ok {
		return value.ClientEmail
	}
	return nil
}

func gteParamIfExists(fl validator.FieldLevel) bool {
	field := fl.Field()
	fieldCompared := fl.Param()

	//omitEmpty
	if field.Kind() == reflect.Int && field.Int() == 0 {
		return true
	}

	var paramFieldValue reflect.Value

	if fl.Parent().Kind() == reflect.Ptr {
		paramFieldValue = fl.Parent().Elem().FieldByName(fieldCompared)
	} else {
		paramFieldValue = fl.Parent().FieldByName(fieldCompared)
	}

	if paramFieldValue.Kind() != reflect.Int {
		return false
	}
	//QUe no rompa cuando quantity no se indica, se romperá en otra validación
	if paramFieldValue.Int() == 0 {
		return true
	}

	if paramFieldValue.Int() > 0 {
		return field.Int() >= paramFieldValue.Int()
	}
	return false
}

func lteParamIfExists(fl validator.FieldLevel) bool {
	field := fl.Field()
	fieldCompared := fl.Param()

	//omitEmpty
	if field.Kind() == reflect.Int && field.Int() == 0 {
		return true
	}

	var paramFieldValue reflect.Value

	if fl.Parent().Kind() == reflect.Ptr {
		paramFieldValue = fl.Parent().Elem().FieldByName(fieldCompared)
	} else {
		paramFieldValue = fl.Parent().FieldByName(fieldCompared)
	}

	if paramFieldValue.Kind() != reflect.Int {
		return false
	}

	if paramFieldValue.Int() == 0 {
		return true
	}

	if paramFieldValue.Int() > 0 {
		return field.Int() <= paramFieldValue.Int()
	}

	return false
}

func requiredIfForBool(fl validator.FieldLevel) bool {
	params := strings.Split(fl.Param(), " ")
	if len(params) != 2 {
		panic(fmt.Sprintf("Bad param number for required_if %s", fl.FieldName()))
	}

	if !requireCheckFieldValue(fl, params[0], params[1], false) {
		return true
	}
	field := fl.Field()
	fl.Parent()
	return field.IsValid() && field.Interface() != reflect.Zero(field.Type()).Interface()
}

func requireCheckFieldValue(fl validator.FieldLevel, param string, value string, defaultNotFoundValue bool) bool {
	field, kind, _, found := fl.GetStructFieldOKAdvanced2(fl.Parent(), param)
	if !found {
		return defaultNotFoundValue
	}

	if kind == reflect.Bool {
		val, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}

		return field.Bool() == val
	}

	return false

}
