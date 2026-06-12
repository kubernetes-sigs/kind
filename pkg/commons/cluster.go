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

var (
	capi_version = "v1.10.10"
	capa_version = "v2.9.3"
	capz_version = "v1.21.3"
	capg_version = "1.6.1-0.4.0"
)

const (
	DefaultDockerhubPrefix = "/dockerhub"
	DefaultEcrpublicPrefix = "/ecrpublic"
	DefaultGhcrPrefix      = "/ghcr"
	DefaultK8sPrefix       = "/k8s"
	DefaultQuayPrefix      = "/quay"
)

type Resource struct {
	APIVersion string      `yaml:"apiVersion" validate:"required"`
	Kind       string      `yaml:"kind" validate:"required"`
	Metadata   Metadata    `yaml:"metadata" validate:"required"`
	Spec       interface{} `yaml:"spec" validate:"required"`
}

type ClusterConfig struct {
	APIVersion string            `yaml:"apiVersion" validate:"required"`
	Kind       string            `yaml:"kind" validate:"required"`
	Metadata   Metadata          `yaml:"metadata" validate:"required"`
	Spec       ClusterConfigSpec `yaml:"spec" validate:"required"`
}

type KeosCluster struct {
	APIVersion string   `yaml:"apiVersion" validate:"required"`
	Kind       string   `yaml:"kind" validate:"required"`
	Metadata   Metadata `yaml:"metadata" validate:"required"`
	Spec       KeosSpec `yaml:"spec" validate:"required"`
}

type Metadata struct {
	Name        string            `yaml:"name,omitempty" validate:"required,min=3,max=100"`
	Namespace   string            `yaml:"namespace,omitempty" `
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type ClusterConfigSpec struct {
	EKSLBController             bool               `yaml:"eks_lb_controller"`
	Private                     bool               `yaml:"private_registry"`
	ControlplaneConfig          ControlplaneConfig `yaml:"controlplane_config"`
	WorkersConfig               WorkersConfig      `yaml:"workers_config"`
	ClusterOperatorVersion      string             `yaml:"cluster_operator_version,omitempty"`
	ClusterOperatorImageVersion string             `yaml:"cluster_operator_image_version,omitempty"`
	PrivateHelmRepo             bool               `yaml:"private_helm_repo"`
	Charts                      []Chart            `yaml:"charts,omitempty"`
	Capx                        CAPX               `yaml:"capx,omitempty"`
	GitOpsEnabled               bool               `yaml:"gitops_enabled"`
}

type CAPX struct {
	CAPI_Version       string `yaml:"capi_version,omitempty"`
	CAPA_Version       string `yaml:"capa_version,omitempty"`
	CAPA_Image_version string `yaml:"capa_image_version,omitempty"`
	CAPG_Version       string `yaml:"capg_version,omitempty"`
	CAPG_Image_version string `yaml:"capg_image_version,omitempty"`
	CAPZ_Version       string `yaml:"capz_version,omitempty"`
	CAPZ_Image_version string `yaml:"capz_image_version,omitempty"`
}

type Chart struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

type ChartEntry struct {
	Repository string
	Version    string
	Namespace  string
	Pull       bool
	Reconcile  bool
}

type ControlplaneConfig struct {
	MaxUnhealthy *int `yaml:"max_unhealthy,omitempty" validate:"omitempty,numeric,gte=0,lte=100"`
}

type WorkersConfig struct {
	MaxUnhealthy *int `yaml:"max_unhealthy,omitempty" validate:"omitempty,numeric,gte=0,lte=100"`
}

type ClusterConfigRef struct {
	Name string `json:"name,omitempty"`
}

// Spec represents the YAML structure in the spec field of the descriptor file
type KeosSpec struct {
	DeployAutoscaler bool `yaml:"deploy_autoscaler" validate:"boolean"`

	Bastion Bastion `yaml:"bastion,omitempty"`

	StorageClass StorageClass `yaml:"storageclass,omitempty"`

	Credentials Credentials `yaml:"credentials,omitempty"`

	InfraProvider string `yaml:"infra_provider" validate:"required,oneof='aws' 'gcp' 'azure'"`

	K8SVersion string `yaml:"k8s_version" validate:"required"`
	Region     string `yaml:"region" validate:"required"`

	Networks Networks `yaml:"networks,omitempty"`

	Dns struct {
		ManageZone bool     `yaml:"manage_zone,omitempty" validate:"boolean"`
		Forwarders []string `yaml:"forwarders,omitempty" validate:"omitempty,dive,ip_addr"`
	} `yaml:"dns,omitempty"`

	DockerRegistries []DockerRegistry `yaml:"docker_registries" validate:"required,dive"`

	HelmRepository HelmRepository `yaml:"helm_repository" validate:"required"`

	ExternalDomain string `yaml:"external_domain" validate:"fqdn"`

	Security Security `yaml:"security,omitempty"`

	Keos Keos `yaml:"keos,omitempty"`

	ControlPlane ControlPlane `yaml:"control_plane" validate:"required,dive"`

	WorkerNodes WorkerNodes `yaml:"worker_nodes" validate:"required,dive"`

	ClusterConfigRef ClusterConfigRef `yaml:"cluster_config_ref,omitempty" validate:"dive"`

	Calico Calico `yaml:"calico,omitempty"`
}

type Calico struct {
	ObservabilityEnabled bool `yaml:"observability_enabled,omitempty" validate:"boolean"`
}

type ControlPlane struct {
	Managed         bool                `yaml:"managed" validate:"boolean"`
	NodeImage       string              `yaml:"node_image,omitempty"`
	HighlyAvailable *bool               `yaml:"highly_available,omitempty" validate:"boolean"`
	Size            string              `yaml:"size,omitempty" validate:"required_if=Managed false"`
	RootVolume      RootVolume          `yaml:"root_volume,omitempty"`
	Tags            []map[string]string `yaml:"tags,omitempty"`
	AWS             AWSCP               `yaml:"aws,omitempty"`
	Azure           AzureCP             `yaml:"azure,omitempty"`
	Gcp             GCPCP               `yaml:"gcp,omitempty"`
	CRIVolume       CustomVolume        `yaml:"cri_volume,omitempty"  validate:"omitempty,dive"`
	ETCDVolume      CustomVolume        `yaml:"etcd_volume,omitempty"  validate:"omitempty,dive"`
	ExtraVolumes    []ExtraVolume       `yaml:"extra_volumes,omitempty" validate:"omitempty,dive"`
}

type GCPCP struct {
	ReleaseChannel                 string                          `yaml:"release_channel"`
	ClusterNetwork                 *ClusterNetwork                 `yaml:"cluster_network,omitempty"`
	MasterAuthorizedNetworksConfig *MasterAuthorizedNetworksConfig `yaml:"master_authorized_networks_config,omitempty"`
	MonitoringConfig               *MonitoringConfig               `yaml:"monitoring_config,omitempty"`
	LoggingConfig                  *LoggingConfig                  `yaml:"logging_config,omitempty"`
	ClusterIpv4Cidr                string                          `yaml:"cluster_ipv4_cidr,omitempty"`
	IPAllocationPolicy             IPAllocationPolicy              `yaml:"ip_allocation_policy,omitempty"`
	ClusterSecurity                *ClusterSecurity                `yaml:"cluster_security,omitempty"`
}

type ClusterNetwork struct {
	PrivateCluster *PrivateCluster `yaml:"private_cluster,omitempty"`
}

type PrivateCluster struct {
	// +kubebuilder:default=true
	EnablePrivateEndpoint *bool `yaml:"enable_private_endpoint,omitempty"`
	// +kubebuilder:default=true
	ControlPlaneCidrBlock string `yaml:"control_plane_cidr_block,omitempty"`
}

// MasterAuthorizedNetworksConfig represents configuration options for master authorized networks feature of the GKE cluster.
type MasterAuthorizedNetworksConfig struct {
	CIDRBlocks []CIDRBlock `yaml:"cidr_blocks,omitempty"`
	// +kubebuilder:default=false
	GCPPublicCIDRsAccessEnabled *bool `yaml:"gcp_public_cidrs_access_enabled,omitempty"`
}

type CIDRBlock struct {
	CIDRBlock string `yaml:"cidr_block"`
	// +kubebuilder:validation:Optional
	DisplayName string `yaml:"display_name,omitempty"`
}

type MonitoringConfig struct {
	// +kubebuilder:default=false
	EnableManagedPrometheus *bool `yaml:"enable_managed_prometheus,omitempty"`
}

type LoggingConfig struct {
	// +kubebuilder:default=false
	SystemComponents *bool `yaml:"system_components,omitempty"`
	// +kubebuilder:default=false
	Workloads *bool `yaml:"workloads,omitempty"`
}

type IPAllocationPolicy struct {
	// +kubebuilder:default=true￼
	UseIPAliases               bool   `yaml:"use_ip_aliases,omitempty"`
	ClusterSecondaryRangeName  string `yaml:"cluster_secondary_range_name,omitempty"`
	ServicesSecondaryRangeName string `yaml:"services_secondary_range_name,omitempty"`
	ClusterIpv4CidrBlock       string `yaml:"cluster_ipv4_cidr_block,omitempty"`
	ServicesIpv4CidrBlock      string `yaml:"services_ipv4_cidr_block,omitempty"`
}

type ClusterSecurity struct {
	WorkloadIdentityConfig *WorkloadIdentityConfig `yaml:"workload_identity_config,omitempty" validate:"required"`
}

type WorkloadIdentityConfig struct {
	// WorkloadPool is the workload pool to attach all Kubernetes service accounts to Google Cloud services.
	// Only relevant when enabled is true
	WorkloadPool string `yaml:"workload_pool,omitempty" validate:"omitempty,workloadpool"`
	// +kubebuilder:default=true
	Enabled *bool `yaml:"enabled,omitempty" validate:"eq=true"`
	// Key: Kubernetes service account name
	// Value: GCP service account email
	ServiceAccounts map[string]string `yaml:"service_accounts,omitempty" validate:"required_if_enabled,gcp_service_accounts"`
}

type Keos struct {
	Flavour string `yaml:"flavour,omitempty"`
}

type Networks struct {
	VPCID         string    `yaml:"vpc_id,omitempty"`
	VPCCIDRBlock  string    `yaml:"vpc_cidr,omitempty" validate:"omitempty,cidrv4"`
	PodsCidrBlock string    `yaml:"pods_cidr,omitempty" validate:"omitempty,cidrv4"`
	PodsSubnets   []Subnets `yaml:"pods_subnets,omitempty" validate:"dive"`
	Subnets       []Subnets `yaml:"subnets,omitempty" validate:"dive"`
	ResourceGroup string    `yaml:"resource_group,omitempty"`
}

type Subnets struct {
	SubnetId  string `yaml:"subnet_id"`
	CidrBlock string `yaml:"cidr,omitempty" validate:"omitempty,cidrv4"`
	Role      string `yaml:"role,omitempty" validate:"omitempty,oneof='control-plane' 'node'"`
}

type AWSCP struct {
	AssociateOIDCProvider bool   `yaml:"associate_oidc_provider,omitempty" validate:"boolean"`
	EncryptionKey         string `yaml:"encryption_key,omitempty"`
	MPRoleName            string `yaml:"mp_role_name,omitempty"`
	Logging               struct {
		ApiServer         bool `yaml:"api_server" validate:"boolean"`
		Audit             bool `yaml:"audit" validate:"boolean"`
		Authenticator     bool `yaml:"authenticator" validate:"boolean"`
		ControllerManager bool `yaml:"controller_manager" validate:"boolean"`
		Scheduler         bool `yaml:"scheduler" validate:"boolean"`
	} `yaml:"logging"`
}

type AzureCP struct {
	Tier string `yaml:"tier" validate:"omitempty,oneof='Free' 'Paid'"`
}

type Security struct {
	ControlPlaneIdentity string `yaml:"control_plane_identity,omitempty"`
	NodesIdentity        string `yaml:"nodes_identity,omitempty"`
	AWS                  struct {
		CreateIAM bool `yaml:"create_iam" validate:"boolean"`
	} `yaml:"aws,omitempty"`
	GCP struct {
		Scopes []string `yaml:"scopes,omitempty"`
	} `yaml:"gcp,omitempty"`
	EnableSecureBoot *bool `yaml:"enable_secure_boot,omitempty"`
}

type WorkerNodes []struct {
	Name             string            `yaml:"name" validate:"required"`
	NodeImage        string            `yaml:"node_image,omitempty"`
	AmiType          string            `yaml:"ami_type,omitempty" validate:"omitempty,oneof=BOTTLEROCKET_x86_64"`
	Quantity         *int              `yaml:"quantity" validate:"required,numeric,gte=0"`
	Size             string            `yaml:"size" validate:"required"`
	ZoneDistribution string            `yaml:"zone_distribution,omitempty" validate:"omitempty,oneof='balanced' 'unbalanced'"`
	AZ               string            `yaml:"az,omitempty"`
	SSHKey           string            `yaml:"ssh_key,omitempty"`
	Spot             bool              `yaml:"spot,omitempty" validate:"boolean"`
	Labels           map[string]string `yaml:"labels,omitempty"`
	AdditionalLabels map[string]string `yaml:"additional_labels,omitempty"`
	Taints           []string          `yaml:"taints,omitempty"`
	NodeGroupMaxSize int               `yaml:"max_size,omitempty" validate:"omitempty,required_with=NodeGroupMinSize,numeric"`
	NodeGroupMinSize *int              `yaml:"min_size,omitempty" validate:"omitempty,required_with=NodeGroupMaxSize,numeric,gte=0"`
	RootVolume       RootVolume        `yaml:"root_volume,omitempty"`
	CRIVolume        CustomVolume      `yaml:"cri_volume,omitempty"  validate:"omitempty,dive"`
	ExtraVolumes     []ExtraVolume     `yaml:"extra_volumes,omitempty" validate:"dive"`
}

// Bastion represents the bastion VM
type Bastion struct {
	NodeImage         string   `yaml:"node_image"`
	VMSize            string   `yaml:"vm_size"`
	AllowedCIDRBlocks []string `yaml:"allowedCIDRBlocks"`
	SSHKey            string   `yaml:"ssh_key"`
}

type RootVolume struct {
	Size          int    `yaml:"size,omitempty"`
	Type          string `yaml:"type,omitempty"`
	Encrypted     bool   `yaml:"encrypted,omitempty"`
	EncryptionKey string `yaml:"encryption_key,omitempty"`
}

type ExtraVolume struct {
	Name          string `yaml:"name,omitempty"`
	DeviceName    string `yaml:"device_name,omitempty"`
	Size          int    `yaml:"size,omitempty"`
	Type          string `yaml:"type,omitempty"`
	Label         string `yaml:"label,omitempty"`
	Encrypted     bool   `yaml:"encrypted,omitempty" validate:"boolean"`
	EncryptionKey string `yaml:"encryption_key,omitempty"`
	MountPath     string `yaml:"mount_path,omitempty"`
}

type CustomVolume struct {
	Enabled       *bool  `yaml:"enabled,omitempty"`
	Size          int    `yaml:"size,omitempty"`
	Type          string `yaml:"type,omitempty"`
	Encrypted     bool   `yaml:"encrypted,omitempty"`
	EncryptionKey string `yaml:"encryption_key,omitempty"`
}

type ETCDVolume struct {
	Size          int    `yaml:"size" validate:"required,numeric"`
	Type          string `yaml:"type,omitempty"`
	Label         string `yaml:"label" validate:"required"`
	Encrypted     bool   `yaml:"encrypted,omitempty" validate:"boolean"`
	EncryptionKey string `yaml:"encryption_key,omitempty"`
}

type CRIVolume struct {
	Size          int    `yaml:"size" validate:"required,numeric"`
	Type          string `yaml:"type,omitempty"`
	Label         string `yaml:"label" validate:"required"`
	Encrypted     bool   `yaml:"encrypted,omitempty" validate:"boolean"`
	EncryptionKey string `yaml:"encryption_key,omitempty"`
}

type ClusterCredentials struct {
	ProviderCredentials         map[string]string
	KeosRegistryCredentials     map[string]string
	DockerRegistriesCredentials []map[string]interface{}
	HelmRepositoryCredentials   map[string]string
	GithubToken                 string
}

type Credentials struct {
	AWS              AWSCredentials              `yaml:"aws" validate:"excluded_with=AZURE GCP"`
	AZURE            AzureCredentials            `yaml:"azure" validate:"excluded_with=AWS GCP"`
	GCP              GCPCredentials              `yaml:"gcp" validate:"excluded_with=AWS AZURE"`
	GithubToken      string                      `yaml:"github_token"`
	DockerRegistries []DockerRegistryCredentials `yaml:"docker_registries"`
	HelmRepository   HelmRepositoryCredentials   `yaml:"helm_repository"`
}

type AWSCredentials struct {
	AccessKey string  `yaml:"access_key"`
	SecretKey string  `yaml:"secret_key"`
	Region    string  `yaml:"region"`
	AccountID string  `yaml:"account_id"`
	RoleARN   *string `yaml:"role_arn,omitempty"`
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
	AuthRequired         bool   `yaml:"auth_required" validate:"boolean"`
	Type                 string `yaml:"type" validate:"required,oneof='acr' 'ecr' 'gar' 'gcr' 'generic'"`
	URL                  string `yaml:"url" validate:"required"`
	KeosRegistry         bool   `yaml:"keos_registry" validate:"boolean"`
	ECRPullThroughCacheEnabled bool   `yaml:"ecr_pull_through_cache_enabled" validate:"boolean"`
}

type HelmRepositoryCredentials struct {
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type HelmRepository struct {
	AuthRequired          bool   `yaml:"auth_required" validate:"boolean"`
	URL                   string `yaml:"url" validate:"required"`
	Type                  string `yaml:"type,omitempty" validate:"oneof='ecr' 'acr' 'gar' 'generic'"`
	ReleaseInterval       string `yaml:"release_interval,omitempty"`
	ReleaseRetries        *int   `yaml:"release_retries,omitempty"`
	ReleaseSourceInterval string `yaml:"release_source_interval,omitempty"`
	RepositoryInterval    string `yaml:"repository_interval,omitempty"`
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
	DockerRegistry   DockerRegistryCredentials   `yaml:"docker_registry"`
	DockerRegistries []DockerRegistryCredentials `yaml:"docker_registries"`
	HelmRepository   HelmRepositoryCredentials   `yaml:"helm_repository"`
}

type EFS struct {
	Name        string `yaml:"name" validate:"required_with=ID"`
	ID          string `yaml:"id" validate:"required_with=Name"`
	Permissions string `yaml:"permissions,omitempty"`
}

type StorageClass struct {
	EncryptionKey string       `yaml:"encryptionKey,omitempty"`
	Class         string       `yaml:"class,omitempty" validate:"omitempty,oneof='standard' 'premium'"`
	Parameters    SCParameters `yaml:"parameters,omitempty"`
}

type SCParameters struct {
	// Common
	Type   string `yaml:"type,omitempty"`
	FsType string `yaml:"fsType,omitempty"`
	Labels string `yaml:"labels,omitempty"`

	// AWS
	AllowAutoIOPSPerGBIncrease string `yaml:"allowAutoIOPSPerGBIncrease,omitempty" validate:"omitempty,oneof='true' 'false'"`
	BlockExpress               string `yaml:"blockExpress,omitempty" validate:"omitempty,oneof='true' 'false'"`
	BlockSize                  string `yaml:"blockSize,omitempty"`
	Iops                       string `yaml:"iops,omitempty" validate:"omitempty,excluded_with=IopsPerGB"`
	IopsPerGB                  string `yaml:"iopsPerGB,omitempty" validate:"omitempty,excluded_with=Iops"`
	Encrypted                  string `yaml:"encrypted,omitempty" validate:"omitempty,oneof='true' 'false'"`
	KmsKeyId                   string `yaml:"kmsKeyId,omitempty"`
	Throughput                 int    `yaml:"throughput,omitempty" validate:"omitempty,gt=0"`

	// Azure
	CachingMode           string `yaml:"cachingMode,omitempty" validate:"omitempty,oneof='None' 'ReadOnly'"`
	DiskAccessID          string `yaml:"diskAccessID,omitempty"`
	DiskEncryptionSetID   string `yaml:"diskEncryptionSetID,omitempty"`
	DiskEncryptionType    string `yaml:"diskEncryptionType,omitempty" validate:"omitempty,oneof='EncryptionAtRestWithCustomerKey' 'EncryptionAtRestWithPlatformAndCustomerKeys'"`
	EnableBursting        string `yaml:"enableBursting,omitempty" validate:"omitempty,oneof='true' 'false'"`
	EnablePerformancePlus string `yaml:"enablePerformancePlus,omitempty" validate:"omitempty,oneof='true' 'false'"`
	Kind                  string `yaml:"kind,omitempty" validate:"omitempty,oneof='managed'"`
	NetworkAccessPolicy   string `yaml:"networkAccessPolicy,omitempty" validate:"omitempty,oneof='AllowAll' 'DenyAll' 'AllowPrivate'"`
	Provisioner           string `yaml:"provisioner,omitempty" validate:"omitempty,oneof='disk.csi.azure.com' 'file.csi.azure.com"`
	PublicNetworkAccess   string `yaml:"publicNetworkAccess,omitempty" validate:"omitempty,oneof='Enabled' 'Disabled'"`
	ResourceGroup         string `yaml:"resourceGroup,omitempty"`
	SkuName               string `yaml:"skuName,omitempty"`
	SubscriptionID        string `yaml:"subscriptionID,omitempty"`
	Tags                  string `yaml:"tags,omitempty"`

	// GCP
	DiskEncryptionKmsKey          string `yaml:"disk-encryption-kms-key,omitempty"`
	ProvisionedIopsOnCreate       string `yaml:"provisioned-iops-on-create,omitempty"`
	ProvisionedThroughputOnCreate string `yaml:"provisioned-throughput-on-create,omitempty"`
	ReplicationType               string `yaml:"replication-type,omitempty"`
}

func (s ClusterConfigSpec) Init() ClusterConfigSpec {
	// Set private registry and helm repository as true by default
	s.Private = true
	s.PrivateHelmRepo = true
	// Set workers config max unhealthy to 100 by default
	s.WorkersConfig.MaxUnhealthy = ToPtr[int](100)
	// Set Git Ops as false by default
	s.GitOpsEnabled = false

	return s
}

func (s ClusterConfigSpec) InitCapx() ClusterConfigSpec {

	setDefaultValue(&s.Capx.CAPI_Version, capi_version)
	setDefaultValue(&s.Capx.CAPA_Version, capa_version)
	setDefaultValue(&s.Capx.CAPA_Image_version, s.Capx.CAPA_Version)
	setDefaultValue(&s.Capx.CAPZ_Version, capz_version)
	setDefaultValue(&s.Capx.CAPZ_Image_version, s.Capx.CAPZ_Version)
	setDefaultValue(&s.Capx.CAPG_Version, capg_version)
	setDefaultValue(&s.Capx.CAPG_Image_version, s.Capx.CAPG_Version)

	return s
}

func setDefaultValue(s *string, value string) {
	if *s == "" {
		*s = value
	}
}

// Init sets default values for the Spec
func (s KeosSpec) Init() KeosSpec {

	highlyAvailable := true
	s.ControlPlane.HighlyAvailable = &highlyAvailable

	// AKS
	s.ControlPlane.Azure.Tier = "Paid"

	// Autoscaler
	s.DeployAutoscaler = true

	// EKS
	s.Security.AWS.CreateIAM = false
	s.ControlPlane.AWS.AssociateOIDCProvider = true
	s.ControlPlane.AWS.Logging.ApiServer = false
	s.ControlPlane.AWS.Logging.Audit = false
	s.ControlPlane.AWS.Logging.Authenticator = false
	s.ControlPlane.AWS.Logging.ControllerManager = false
	s.ControlPlane.AWS.Logging.Scheduler = false

	// INIT GKE
	s.ControlPlane.Gcp.ReleaseChannel = "extended"
	// Enable secure boot by default
	// Only enable secure boot by default for GCP
	if s.InfraProvider == "gcp" && s.Security.EnableSecureBoot == nil {
		s.Security.EnableSecureBoot = ToPtr(true)
	}
	if s.ControlPlane.Gcp.ClusterNetwork == nil {
		s.ControlPlane.Gcp.ClusterNetwork = &ClusterNetwork{}
	}
	if s.ControlPlane.Gcp.ClusterNetwork.PrivateCluster == nil {
		s.ControlPlane.Gcp.ClusterNetwork.PrivateCluster = &PrivateCluster{}
	}
	if s.ControlPlane.Gcp.ClusterNetwork.PrivateCluster.EnablePrivateEndpoint == nil {
		s.ControlPlane.Gcp.ClusterNetwork.PrivateCluster.EnablePrivateEndpoint = ToPtr(true)
	}
	if s.ControlPlane.Gcp.MasterAuthorizedNetworksConfig == nil {
		s.ControlPlane.Gcp.MasterAuthorizedNetworksConfig = &MasterAuthorizedNetworksConfig{}
	}
	if s.ControlPlane.Gcp.MasterAuthorizedNetworksConfig.GCPPublicCIDRsAccessEnabled == nil {
		s.ControlPlane.Gcp.MasterAuthorizedNetworksConfig.GCPPublicCIDRsAccessEnabled = ToPtr(false)
	}
	if s.ControlPlane.Gcp.MonitoringConfig == nil {
		s.ControlPlane.Gcp.MonitoringConfig = &MonitoringConfig{}
	}
	if s.ControlPlane.Gcp.LoggingConfig == nil {
		s.ControlPlane.Gcp.MonitoringConfig.EnableManagedPrometheus = ToPtr(false)
	}
	if s.ControlPlane.Gcp.LoggingConfig == nil {
		s.ControlPlane.Gcp.LoggingConfig = &LoggingConfig{}
	}
	if s.ControlPlane.Gcp.LoggingConfig.SystemComponents == nil {
		s.ControlPlane.Gcp.LoggingConfig.SystemComponents = ToPtr(false)
		s.ControlPlane.Gcp.LoggingConfig.Workloads = ToPtr(false)
	}
	// END GKE

	// Helm
	s.HelmRepository.AuthRequired = false
	s.HelmRepository.Type = "generic"

	// Managed zones
	s.Dns.ManageZone = true

	return s
}

// GetPrefixedRegistryURL returns the registry URL with prefix if appropriate
func GetPrefixedRegistryURL(originalRegistry string, baseRegistryURL string, ecrPullThroughCacheEnabled bool) string {
	if !ecrPullThroughCacheEnabled || baseRegistryURL == "" {
		return baseRegistryURL
	}

	prefix := ""
	switch {
	case strings.Contains(originalRegistry, "docker.io"):
		prefix = DefaultDockerhubPrefix
	case strings.Contains(originalRegistry, "public.ecr.aws"):
		prefix = DefaultEcrpublicPrefix
	case strings.Contains(originalRegistry, "ghcr.io"):
		prefix = DefaultGhcrPrefix
	case strings.Contains(originalRegistry, "quay.io"):
		prefix = DefaultQuayPrefix
	case strings.Contains(originalRegistry, "k8s.io"):
		prefix = DefaultK8sPrefix
	}

	return baseRegistryURL + prefix
}

// Validator for WorkloadPool field
func workloadPoolValidator(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if value == "" {
		return true // omitempty
	}

	// Must match "<projectid>.svc.id.goog"
	parts := strings.Split(value, ".svc.id.goog")
	if len(parts) != 2 || parts[0] == "" {
		fmt.Printf("DEBUG workloadPool: formato inválido '%s'\n", value)
		return false
	}
	return true
}

// Validator for required_if_enabled
func requiredIfEnabledValidator(fl validator.FieldLevel) bool {
	parent := fl.Parent()
	enabledField := parent.FieldByName("Enabled")

	if !enabledField.IsValid() || enabledField.IsNil() {
		return true
	}

	enabled := enabledField.Elem().Bool()
	if !enabled {
		return true
	}

	field := fl.Field()

	if !field.IsValid() || (field.Kind() == reflect.Ptr && field.IsNil()) {
		return false
	}

	switch field.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() > 0
	default:
		return field.Interface() != reflect.Zero(field.Type()).Interface()
	}
}

// Validator for ServiceAccounts field
func gcpServiceAccountsValidator(fl validator.FieldLevel) bool {
	serviceAccounts, ok := fl.Field().Interface().(map[string]string)
	if !ok {
		return false
	}

	// Get parent struct (WorkloadIdentityConfig)
	parent := fl.Parent()
	workloadPoolField := parent.FieldByName("WorkloadPool")
	if !workloadPoolField.IsValid() {
		return false
	}
	workloadPool := workloadPoolField.String()

	var poolProjectID string
	if workloadPool != "" {
		parts := strings.Split(workloadPool, ".svc.id.goog")
		if len(parts) == 2 && parts[0] != "" {
			poolProjectID = parts[0]
		}
	}

	for _, saEmail := range serviceAccounts {
		// 1. Format check
		if !strings.HasSuffix(saEmail, ".iam.gserviceaccount.com") {
			return false
		}
		parts := strings.Split(saEmail, "@")
		if len(parts) != 2 {
			return false
		}

		// 2. Consistency check
		if poolProjectID != "" {
			// Extract project ID from email: <sa-name>@<project-id>.iam.gserviceaccount.com
			domainParts := strings.Split(parts[1], ".iam.gserviceaccount.com")
			if len(domainParts) < 1 {
				return false
			}
			saProjectID := domainParts[0]

			if saProjectID != poolProjectID {
				fmt.Printf("ERROR: SA MISMATCH - saProjectID='%s' not equal to poolProjectID='%s'\n", saProjectID, poolProjectID)
				return false
			}
		}
	}

	return true
}

// Read descriptor file
func GetClusterDescriptor(descriptorPath string) (*KeosCluster, *ClusterConfig, error) {
	var keosCluster KeosCluster
	var clusterConfig ClusterConfig
	findClusterConfig := false

	_, err := os.Stat(descriptorPath)
	if err != nil {
		return nil, nil, errors.New("No exists any cluster descriptor as " + descriptorPath)
	}

	descriptorRAW, err := os.ReadFile(descriptorPath)
	if err != nil {
		return nil, nil, err
	}

	validate := validator.New()
	validate.RegisterValidation("gte_param_if_exists", gteParamIfExists)
	validate.RegisterValidation("lte_param_if_exists", lteParamIfExists)
	validate.RegisterValidation("required_if_for_bool", requiredIfForBool)
	validate.RegisterValidation("required_if_enabled", requiredIfEnabledValidator)
	validate.RegisterValidation("workloadpool", workloadPoolValidator)
	validate.RegisterValidation("gcp_service_accounts", gcpServiceAccountsValidator)

	descriptorManifests := strings.Split(string(descriptorRAW), "---\n")
	for _, manifest := range descriptorManifests {
		var resource Resource
		err = yaml.Unmarshal([]byte(manifest), &resource)
		if err != nil {
			return nil, nil, err
		}
		if !reflect.DeepEqual(resource, Resource{}) {
			err = validate.Struct(resource)
			if err != nil {
				return nil, nil, err
			}

			switch resource.Kind {
			case "KeosCluster":
				keosCluster.Spec = new(KeosSpec).Init()
				err = yaml.Unmarshal([]byte(manifest), &keosCluster)
				if err != nil {
					return nil, nil, err
				}

				// If WorkloadPool is not set, but workload identity is enabled, set default value based on GCP ProjectID from credentials
				if keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity != nil &&
					keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig != nil &&
					keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.WorkloadPool == "" &&
					keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.Enabled != nil &&
					*keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.Enabled &&
					keosCluster.Spec.Credentials.GCP.ProjectID != "" {
					// NOTE: We do not need to check if ProjectID is empty, because validation will fail if GCP credentials are not set properly
					keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.WorkloadPool = keosCluster.Spec.Credentials.GCP.ProjectID + ".svc.id.goog"
				}

				err = validate.Struct(keosCluster)
				if err != nil {
					// Si el error es por workload_pool, muestra un mensaje más claro
					if validationErrors, ok := err.(validator.ValidationErrors); ok {
						for _, ve := range validationErrors {
							if ve.StructNamespace() == "KeosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.WorkloadPool" &&
								ve.Tag() == "workloadpool" {
								return nil, nil, fmt.Errorf(
									"ERROR: The 'workload_pool' field in 'workload_identity_config' is invalid.\n" +
										"It must have the format: <projectid>.svc.id.goog (example: clusterapi-371111.svc.id.goog)\n")
							}
							if ve.StructNamespace() == "KeosCluster.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig" &&
								ve.Tag() == "required" &&
								keosCluster.Spec.ControlPlane.Gcp.ClusterSecurity != nil {
								return nil, nil, fmt.Errorf(
									"ERROR: Invalid format in 'cluster_security'.\n" +
										"The 'workload_identity_config' field is missing. Ensure the structure is:\n" +
										"cluster_security:\n" +
										"  workload_identity_config:\n" +
										"    enabled: true\n" +
										"    ...\n")
							}
							if ve.Tag() == "gcp_service_accounts" {
								return nil, nil, fmt.Errorf(
									"ERROR: The service accounts in 'service_accounts' are invalid.\n" +
										"They must follow the format: <name>@<project_id>.iam.gserviceaccount.com\n" +
										"And the <project_id> must match the one defined in 'workload_pool'.\n")
							}
							if ve.Tag() == "required_if_enabled" {
								return nil, nil, fmt.Errorf(
									"ERROR: 'service_accounts' is required when 'enabled' is true in 'workload_identity_config'.\n" +
										"Add at least one GCP service account (format: <name>@<project-id>.iam.gserviceaccount.com)\n")
							}
						}
					}
					return nil, nil, err
				}

				keosCluster.Metadata.Namespace = "cluster-" + keosCluster.Metadata.Name
			case "ClusterConfig":
				findClusterConfig = true
				clusterConfig.Spec = new(ClusterConfigSpec).Init()
				err = yaml.Unmarshal([]byte(manifest), &clusterConfig)
				if err != nil {
					return nil, nil, err
				}
				err = validate.Struct(clusterConfig)
				if err != nil {
					return nil, nil, err
				}
				clusterConfig.Metadata.Namespace = "cluster-" + keosCluster.Metadata.Name
			default:
				return nil, nil, errors.New("Unsupported manifest kind: " + resource.Kind)
			}
		}
	}

	if reflect.DeepEqual(keosCluster, KeosCluster{}) {
		return nil, nil, errors.New("Keoscluster's manifest has not been found.")
	}

	if !findClusterConfig {
		clusterConfig = ClusterConfig{}
		clusterConfig.APIVersion = "installer.stratio.com/v1beta1"
		clusterConfig.Kind = "ClusterConfig"
		clusterConfig.Metadata.Name = keosCluster.Spec.InfraProvider + "-config"
		clusterConfig.Metadata.Namespace = "cluster-" + keosCluster.Metadata.Name
		clusterConfig.Spec = new(ClusterConfigSpec).Init()
		if !keosCluster.Spec.ControlPlane.Managed {
			clusterConfig.Spec.ControlplaneConfig.MaxUnhealthy = ToPtr[int](34)
		}
	}

	clusterConfig.Spec = clusterConfig.Spec.InitCapx()

	// Clean unnecessary fields for keosCluster before processing
	CleanKeosClusterBeforeInstall(&keosCluster)

	return &keosCluster, &clusterConfig, nil
}

func CleanKeosClusterBeforeInstall(kc *KeosCluster) {
	if kc.Spec.ControlPlane.Gcp.ClusterSecurity != nil &&
		kc.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig != nil {
		kc.Spec.ControlPlane.Gcp.ClusterSecurity.WorkloadIdentityConfig.ServiceAccounts = nil
	}

	// Add more fields here if you need to clean others in the future
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

func gteParamIfExists(fl validator.FieldLevel) bool {
	field := fl.Field()
	fieldCompared := fl.Param()

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

// Ptr returns a pointer to the provided value.
func ToPtr[T any](v T) *T {
	return &v
}
