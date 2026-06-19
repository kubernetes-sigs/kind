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
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed templates/*/*
var ctel embed.FS

//go:embed files/*/deny-all-egress-imds_gnetpol.yaml
var denyAllEgressIMDSgnpFiles embed.FS

//go:embed files/*/allow-egress-imds_gnetpol.yaml
var allowEgressIMDSgnpFiles embed.FS

//go:embed files/azure/flux2_azurepodidentityexception.yaml
var azureFlux2PodIdentityException string

var stratio_helm_repo string

//go:embed files/*/*_pdb.yaml
var commonsPDBFile embed.FS

const (
	CAPICoreProvider         = "cluster-api"
	CAPIBootstrapProvider    = "kubeadm"
	CAPIControlPlaneProvider = "kubeadm"
	// CAPIVersion              = "v1.10.8"

	scName = "keos"

	postInstallAnnotation = "cluster-autoscaler.kubernetes.io/safe-to-evict-local-volumes"
	corednsPdbPath        = "/kind/coredns_pdb.yaml"

	machineHealthCheckWorkerNodePath       = "/kind/manifests/machinehealthcheckworkernode.yaml"
	machineHealthCheckControlPlaneNodePath = "/kind/manifests/machinehealthcheckcontrolplane.yaml"
	defaultScAnnotation                    = "storageclass.kubernetes.io/is-default-class"
)

//go:embed files/common/calico-metrics.yaml
var calicoMetrics string

type ChartsDictionary struct {
	Charts map[string]map[string]map[string]commons.ChartEntry
}

type PrivateParams struct {
	KeosCluster commons.KeosCluster
	KeosRegUrl  string
	CentralECR  bool
	Private     bool
	HelmPrivate bool
}

type PBuilder interface {
	setCapx(managed bool, capx commons.CAPX)
	setCapxEnvVars(p ProviderParams)
	setSC(p ProviderParams)
	pullProviderCharts(n nodes.Node, clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterCredentials commons.ClusterCredentials, clusterType string) error
	getProviderCharts(clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterType string) map[string]commons.ChartEntry
	getOverriddenCharts(charts *[]commons.Chart, clusterConfigSpec *commons.ClusterConfigSpec, clusterType string) []commons.Chart
	installCloudProvider(n nodes.Node, k string, privateParams PrivateParams) error
	installCSI(n nodes.Node, k string, privateParams PrivateParams, providerParams ProviderParams, chartsList map[string]commons.ChartEntry) error
	getProvider() Provider
	configureStorageClass(n nodes.Node, k string) error
	internalNginx(p ProviderParams, networks commons.Networks) (bool, error)
	getOverrideVars(p ProviderParams, networks commons.Networks, clusterConfigSpec commons.ClusterConfigSpec) (map[string][]byte, error)
	getRegistryCredentials(p ProviderParams, u string) (string, string, error)
	postInstallPhase(n nodes.Node, k string) error
}

type Provider struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxManaged      bool
	capxName         string
	capxEnvVars      []string
	scParameters     commons.SCParameters
	scProvisioner    string
	csiNamespace     string
}

type Node struct {
	AZ      string
	QA      int
	MaxSize int
	MinSize int
}

type Infra struct {
	builder PBuilder
}

type ProviderParams struct {
	ClusterName  string
	Region       string
	Managed      bool
	Credentials  map[string]string
	GithubToken  string
	StorageClass commons.StorageClass
	Capx         commons.CAPX
}

type DefaultStorageClass struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Name        string            `yaml:"name"`
	} `yaml:"metadata"`
	AllowVolumeExpansion bool                 `yaml:"allowVolumeExpansion"`
	Provisioner          string               `yaml:"provisioner"`
	Parameters           commons.SCParameters `yaml:"parameters"`
	VolumeBindingMode    string               `yaml:"volumeBindingMode"`
}

type helmRepository struct {
	url  string
	user string
	pass string
}

type calicoHelmParams struct {
	Spec           commons.KeosSpec
	KeosRegUrl     string
	QuayRegUrl     string
	Private        bool
	IsNetPolEngine bool
	Annotations    map[string]string
}

type commonHelmParams struct {
	KeosRegUrl string
	Private    bool
	CentralECR bool
}

type cloudControllerHelmParams struct {
	ClusterName string
	Private     bool
	KeosRegUrl  string
	PodsCidr    string
}

type fluxHelmRepositoryParams struct {
	ChartName          string
	ChartRepoUrl       string
	ChartRepoScheme    string
	Spec               commons.KeosSpec
	HelmRepoCreds      HelmRegistry
	RepositoryInterval string
}

type fluxHelmReleaseParams struct {
	HelmReleaseName string
	ChartName       string
	ChartNamespace  string
	ChartRepoRef    string
	ChartVersion    string
}

var scTemplate = DefaultStorageClass{
	APIVersion: "storage.k8s.io/v1",
	Kind:       "StorageClass",
	Metadata: struct {
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Name        string            `yaml:"name"`
	}{
		Annotations: map[string]string{
			defaultScAnnotation: "true",
		},
		Name: scName,
	},
	AllowVolumeExpansion: true,
	VolumeBindingMode:    "WaitForFirstConsumer",
}

var commonsCharts = ChartsDictionary{
	Charts: map[string]map[string]map[string]commons.ChartEntry{
		"32": {
			"managed": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
			"unmanaged": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
		},
		"33": {
			"managed": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
			"unmanaged": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
		},
		"34": {
			"managed": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
			"unmanaged": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.17.2", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
		},
		"35": {
			"managed": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.18.4", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
			"unmanaged": {
				"cert-manager": {Repository: "https://charts.jetstack.io", Version: "v1.20.2", Namespace: "cert-manager", Pull: true, Reconcile: true},
				"flux2":        {Repository: "https://fluxcd-community.github.io/helm-charts", Version: "2.18.4", Namespace: "kube-system", Pull: true, Reconcile: true},
			},
		},
	},
}

func getBuilder(builderType string) PBuilder {
	if builderType == "aws" {
		return newAWSBuilder()
	}

	if builderType == "gcp" {
		return newGCPBuilder()
	}

	if builderType == "azure" {
		return newAzureBuilder()
	}
	return nil
}

func newInfra(b PBuilder) *Infra {
	return &Infra{
		builder: b,
	}
}

func (i *Infra) buildProvider(p ProviderParams) Provider {
	i.builder.setCapx(p.Managed, p.Capx)
	i.builder.setCapxEnvVars(p)
	i.builder.setSC(p)
	return i.builder.getProvider()
}

func (i *Infra) getProviderCharts(clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec) map[string]commons.ChartEntry {
	clusterType := "managed"
	if !keosSpec.ControlPlane.Managed {
		clusterType = "unmanaged"
	}

	commonsChartsList := getGenericCharts(clusterConfigSpec, keosSpec, commonsCharts, clusterType)

	providerChartsList := i.builder.getProviderCharts(clusterConfigSpec, keosSpec, clusterType)

	completedChartsList := make(map[string]commons.ChartEntry)
	for key, value := range commonsChartsList {
		completedChartsList[key] = value
	}
	for key, value := range providerChartsList {
		completedChartsList[key] = value
	}

	return completedChartsList
}

func (i *Infra) pullProviderCharts(n nodes.Node, clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterCredentials commons.ClusterCredentials) error {
	clusterType := "managed"
	if !keosSpec.ControlPlane.Managed {
		clusterType = "unmanaged"
	}

	if err := pullGenericCharts(n, clusterConfigSpec, keosSpec, clusterCredentials, commonsCharts, clusterType); err != nil {
		return err
	}

	if err := i.builder.pullProviderCharts(n, clusterConfigSpec, keosSpec, clusterCredentials, clusterType); err != nil {
		return err
	}
	clusterConfigSpec.Charts = i.getOverriddenCharts(clusterConfigSpec, clusterType)
	return nil

}

func (i *Infra) getOverriddenCharts(clusterConfigSpec *commons.ClusterConfigSpec, clusterType string) []commons.Chart {
	charts := ConvertToChart(commonsCharts.Charts[majorVersion][clusterType])
	for _, ovChart := range clusterConfigSpec.Charts {
		for _, chart := range *charts {
			if chart.Name == ovChart.Name {
				chart.Version = ovChart.Version
			}
		}
	}
	return i.builder.getOverriddenCharts(charts, clusterConfigSpec, clusterType)
}

func ConvertToChart(chartEntries map[string]commons.ChartEntry) *[]commons.Chart {
	var charts []commons.Chart
	for name, entry := range chartEntries {
		if entry.Pull {
			chart := commons.Chart{
				Name:    name,
				Version: entry.Version,
			}
			charts = append(charts, chart)
		}

	}
	return &charts
}

func getGenericCharts(clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, chartDictionary ChartsDictionary, clusterType string) map[string]commons.ChartEntry {
	chartsToInstall := chartDictionary.Charts[majorVersion][clusterType]
	for _, overrideChart := range clusterConfigSpec.Charts {
		chart := chartsToInstall[overrideChart.Name]
		if !reflect.DeepEqual(chart, commons.ChartEntry{}) {

			chart.Version = overrideChart.Version
			chartsToInstall[overrideChart.Name] = chart
		}
	}
	if clusterConfigSpec.PrivateHelmRepo {
		for name, entry := range chartsToInstall {
			entry.Repository = keosSpec.HelmRepository.URL
			chartsToInstall[name] = entry
		}
	}
	return chartsToInstall
}

func pullGenericCharts(n nodes.Node, clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterCredentials commons.ClusterCredentials, chartDictionary ChartsDictionary, clusterType string) error {
	chartsToInstall := getGenericCharts(clusterConfigSpec, keosSpec, chartDictionary, clusterType)
	return pullCharts(n, chartsToInstall, keosSpec, clusterCredentials)
}

func (i *Infra) installCloudProvider(n nodes.Node, k string, privateParams PrivateParams) error {
	return i.builder.installCloudProvider(n, k, privateParams)
}

func (i *Infra) installCSI(n nodes.Node, k string, privateParams PrivateParams, providerParams ProviderParams, chartsList map[string]commons.ChartEntry) error {
	return i.builder.installCSI(n, k, privateParams, providerParams, chartsList)
}

func (i *Infra) configureStorageClass(n nodes.Node, k string) error {
	return i.builder.configureStorageClass(n, k)
}

func (i *Infra) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	return i.builder.internalNginx(p, networks)
}

func (i *Infra) getOverrideVars(p ProviderParams, networks commons.Networks, clusterConfigSpec commons.ClusterConfigSpec) (map[string][]byte, error) {
	return i.builder.getOverrideVars(p, networks, clusterConfigSpec)
}

func (i *Infra) getRegistryCredentials(p ProviderParams, u string) (string, string, error) {
	return i.builder.getRegistryCredentials(p, u)
}

func (i *Infra) postInstallPhase(n nodes.Node, k string) error {
	return i.builder.postInstallPhase(n, k)
}

func (p *Provider) getDenyAllEgressIMDSGNetPol() (string, error) {
	denyAllEgressIMDSGNetPolLocalPath := "files/" + p.capxProvider + "/deny-all-egress-imds_gnetpol.yaml"
	denyAllEgressIMDSgnpFile, err := denyAllEgressIMDSgnpFiles.Open(denyAllEgressIMDSGNetPolLocalPath)
	if err != nil {
		return "", errors.Wrap(err, "error opening the deny all egress IMDS file")
	}
	defer denyAllEgressIMDSgnpFile.Close()
	denyAllEgressIMDSgnpContent, err := io.ReadAll(denyAllEgressIMDSgnpFile)
	if err != nil {
		return "", err
	}

	return string(denyAllEgressIMDSgnpContent), nil
}

func (p *Provider) getAllowCAPXEgressIMDSGNetPol() (string, error) {
	allowEgressIMDSGNetPolLocalPath := "files/" + p.capxProvider + "/allow-egress-imds_gnetpol.yaml"
	allowEgressIMDSgnpFile, err := allowEgressIMDSgnpFiles.Open(allowEgressIMDSGNetPolLocalPath)
	if err != nil {
		return "", errors.Wrap(err, "error opening the allow egress IMDS file")
	}
	defer allowEgressIMDSgnpFile.Close()
	allowEgressIMDSgnpContent, err := io.ReadAll(allowEgressIMDSgnpFile)
	if err != nil {
		return "", err
	}

	return string(allowEgressIMDSgnpContent), nil
}

func getcapxPDB(commonsPDBLocalPath string) (string, error) {
	commonsPDBFile, err := commonsPDBFile.Open(commonsPDBLocalPath)
	if err != nil {
		return "", errors.Wrap(err, "error opening the PodDisruptionBudget file")
	}
	defer commonsPDBFile.Close()
	capaPDBContent, err := io.ReadAll(commonsPDBFile)
	if err != nil {
		return "", err
	}

	return string(capaPDBContent), nil
}

func (p *Provider) deployCertManager(n nodes.Node, keosRegistryUrl string, kubeconfigPath string, privateParams PrivateParams, chartsList map[string]commons.ChartEntry) error {
	certManagerValuesFile := "/kind/cert-manager-helm-values.yaml"
	certManagerHelmParams := commonHelmParams{
		KeosRegUrl: keosRegistryUrl,
		Private:    privateParams.Private,
		CentralECR: privateParams.CentralECR,
	}

	// if central ecr is enabled add suffix to the image
	if privateParams.CentralECR {
		certManagerHelmParams.KeosRegUrl = commons.GetPrefixedRegistryURL("quay.io", privateParams.KeosRegUrl, privateParams.CentralECR)
	}
	certManagerHelmValues, err := getManifest("common", "cert-manager-helm-values.tmpl", majorVersion, certManagerHelmParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate cert-manager helm values")
	}

	c := "echo '" + certManagerHelmValues + "' > " + certManagerValuesFile
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create cert-manager Helm chart values file")
	}
	c = "helm install --wait cert-manager /stratio/helm/cert-manager" +
		" --namespace=cert-manager" +
		" --create-namespace" +
		" --values " + certManagerValuesFile
	if kubeconfigPath != "" {
		c = c + " --kubeconfig " + kubeconfigPath
	}
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy cert-manager Helm Chart")
	}
	return nil
}

func (p *Provider) deployClusterOperator(n nodes.Node, privateParams PrivateParams, clusterCredentials commons.ClusterCredentials, keosRegistry KeosRegistry, clusterConfig *commons.ClusterConfig, kubeconfigPath string, firstInstallation bool, helmRepoCreds HelmRegistry) error {
	var c string
	var err error
	var helmRepository helmRepository
	var chartVersion string
	clusterOperatorImage := ""
	keosCluster := privateParams.KeosCluster

	if clusterConfig != nil {
		if clusterConfig.Spec.ClusterOperatorVersion != "" {
			chartVersion = clusterConfig.Spec.ClusterOperatorVersion
		} else {
			chartVersion, err = getLastChartVersion(helmRepoCreds)
			if err != nil {
				return errors.Wrap(err, "failed to get the last chart version")
			}
			clusterConfig.Spec.ClusterOperatorVersion = chartVersion
		}
		if clusterConfig.Spec.ClusterOperatorImageVersion != "" {
			clusterOperatorImage = clusterConfig.Spec.ClusterOperatorImageVersion
		}
	}

	if firstInstallation && keosCluster.Spec.InfraProvider == "aws" && strings.HasPrefix(keosCluster.Spec.HelmRepository.URL, "s3://") {
		c = "mkdir -p ~/.aws"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create aws config file")
		}
		c = "echo [default] > ~/.aws/config && " +
			"echo region = " + keosCluster.Spec.Region + " >>  ~/.aws/config"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create aws config file")
		}
		awsCredentials := "[default]\naws_access_key_id = " + clusterCredentials.ProviderCredentials["AccessKey"] + "\naws_secret_access_key = " + clusterCredentials.ProviderCredentials["SecretKey"] + "\n"
		c = "echo '" + awsCredentials + "' > ~/.aws/credentials"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create aws credentials file")
		}
	}

	// Create the docker registries credentials secret for keoscluster-controller-manager
	if clusterCredentials.DockerRegistriesCredentials != nil && firstInstallation {
		jsonDockerRegistriesCredentials, err := json.Marshal(clusterCredentials.DockerRegistriesCredentials)
		if err != nil {
			return errors.Wrap(err, "failed to marshal docker registries credentials")
		}
		c = "kubectl -n kube-system create secret generic keoscluster-registries --from-literal=credentials='" + string(jsonDockerRegistriesCredentials) + "'"
		if kubeconfigPath != "" {
			c = c + " --kubeconfig " + kubeconfigPath
		}
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create keoscluster-registries secret")
		}
	}

	if kubeconfigPath == "" {
		// Clean keoscluster file
		keosCluster.Spec.Credentials = commons.Credentials{}
		keosCluster.Spec.StorageClass = commons.StorageClass{}
		keosCluster.Spec.Security.AWS = struct {
			CreateIAM bool "yaml:\"create_iam\" validate:\"boolean\""
		}{}
		if keosCluster.Spec.InfraProvider != "azure" || (keosCluster.Spec.InfraProvider == "azure" && !keosCluster.Spec.ControlPlane.Managed) {
			keosCluster.Spec.ControlPlane.Azure = commons.AzureCP{}
		}
		if keosCluster.Spec.InfraProvider != "aws" || (keosCluster.Spec.InfraProvider == "aws" && !keosCluster.Spec.ControlPlane.Managed) {
			keosCluster.Spec.ControlPlane.AWS = commons.AWSCP{}
		}
		if !(keosCluster.Spec.InfraProvider == "gcp" && keosCluster.Spec.ControlPlane.Managed) {
			keosCluster.Spec.ControlPlane.Gcp = commons.GCPCP{}
		}

		if keosCluster.Spec.ControlPlane.Managed {
			keosCluster.Spec.ControlPlane.HighlyAvailable = nil
		}
		keosCluster.Spec.Keos = commons.Keos{}

		clusterConfigYAML, err := yaml.Marshal(clusterConfig)
		if err != nil {
			return err
		}
		// Write keoscluster file
		c = "echo '" + string(clusterConfigYAML) + "' > " + manifestsPath + "/clusterconfig.yaml"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to write the keoscluster file")
		}
		keosCluster.Spec.ClusterConfigRef.Name = clusterConfig.Metadata.Name

		keosClusterYAML, err := yaml.Marshal(keosCluster)
		if err != nil {
			return err
		}
		// Write keoscluster file
		c = "echo '" + string(keosClusterYAML) + "' > " + manifestsPath + "/keoscluster.yaml"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to write the keoscluster file")
		}
		// Add helm repository
		helmRepository.url = keosCluster.Spec.HelmRepository.URL
		if strings.HasPrefix(helmRepository.url, "oci://") {
			stratio_helm_repo = helmRepoCreds.URL
		} else {
			stratio_helm_repo = "stratio-helm-repo"
		}

		if firstInstallation {
			// Pull cluster-operator helm chart
			c = "helm pull " + stratio_helm_repo + "/cluster-operator --version " + chartVersion +
				" --untar --untardir /stratio/helm"
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to pull cluster-operator helm chart")
			}
		}
		// Deploy cluster-operator chart
		c = "helm install --wait cluster-operator /stratio/helm/cluster-operator" +
			" --namespace kube-system" +
			" --set provider=" + keosCluster.Spec.InfraProvider +
			" --set app.containers.controllerManager.image.registry=" + keosRegistry.url +
			" --set app.containers.controllerManager.image.repository=stratio/cluster-operator" +
			" --set app.containers.controllerManager.imagePullSecrets.enabled=true"
		if clusterOperatorImage != "" {
			c += " --set app.containers.controllerManager.image.tag=" + clusterOperatorImage
		}
		if keosCluster.Spec.InfraProvider == "azure" {
			c += " --set secrets.azure.clientIDBase64=" + strings.Split(p.capxEnvVars[1], "AZURE_CLIENT_ID_B64=")[1] +
				" --set secrets.azure.clientSecretBase64=" + strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET_B64=")[1] +
				" --set secrets.azure.subscriptionIDBase64=" + strings.Split(p.capxEnvVars[2], "AZURE_SUBSCRIPTION_ID_B64=")[1] +
				" --set secrets.azure.tenantIDBase64=" + strings.Split(p.capxEnvVars[3], "AZURE_TENANT_ID_B64=")[1]
		} else if keosCluster.Spec.InfraProvider == "gcp" {
			c += " --set secrets.common.credentialsBase64=" + strings.Split(p.capxEnvVars[0], "GCP_B64ENCODED_CREDENTIALS=")[1]
		} else if keosCluster.Spec.InfraProvider == "aws" {
			c += " --set secrets.common.credentialsBase64=" + strings.Split(p.capxEnvVars[3], "AWS_B64ENCODED_CREDENTIALS=")[1]
		}
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to deploy cluster-operator chart")
		}
	} else {
		helmValuesClusterOperatorFile := "/kind/cluster-operator-helm-values.yaml"
		c = "helm get values cluster-operator" +
			" --namespace kube-system --all > " +
			helmValuesClusterOperatorFile
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster-operator helm values file")
		}

		// Read the YAML file
		c = "cat " + helmValuesClusterOperatorFile
		helmValuesClusterOperatorData, err := commons.ExecuteCommand(n, c, 5, 3)
		if err != nil || helmValuesClusterOperatorData == "" {
			return errors.Wrap(err, "failed to read HelmRelease values file")
		}
		// Unmarshal YAML data into a map
		var helmReleaseValues map[string]interface{}
		if err := yaml.Unmarshal([]byte(helmValuesClusterOperatorData), &helmReleaseValues); err != nil {
			return errors.Wrap(err, "failed to unmarshal HelmRelease values file")
		}

		// Convert app field to map[string]interface{}
		appValues, _ := helmReleaseValues["app"].(map[string]interface{})

		// Update the app.replicas
		appValues["replicas"] = 2
		// Update the app.containers.controllerManager.imagePullSecrets.enabled
		nested := appValues["containers"].(map[string]interface{})
		nested = nested["controllerManager"].(map[string]interface{})
		nested = nested["imagePullSecrets"].(map[string]interface{})
		nested["enabled"] = false
		// Update the 'app' field in the original map
		helmReleaseValues["app"] = appValues
		// Marshal the updated data back to YAML
		updatedHelmValuesClusterOperatorData, err := yaml.Marshal(&helmReleaseValues)
		if err != nil {
			return errors.Wrap(err, "failed to marshal updated HelmRelease values content")
		}
		// Write the updated YAML data back to the file
		c = "echo '" + string(updatedHelmValuesClusterOperatorData) + "' > " + helmValuesClusterOperatorFile
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to write updated HelmRelease values file")
		}

		clusterOperatorHelmReleaseParams := fluxHelmReleaseParams{
			HelmReleaseName: "cluster-operator",
			ChartName:       "cluster-operator",
			ChartNamespace:  "kube-system",
			ChartRepoRef:    "keos",
			ChartVersion:    chartVersion,
		}
		// Create Helm release using the fluxHelmReleaseParams
		if err := configureHelmRelease(n, kubeconfigPath, "flux2_helmrelease.tmpl", clusterOperatorHelmReleaseParams, privateParams.KeosCluster.Spec.HelmRepository); err != nil {
			return err
		}
	}

	// Wait for cluster-operator deployment
	c = "kubectl -n kube-system rollout status deploy/keoscluster-controller-manager --timeout=3m"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to wait for cluster-operator deployment")
	}

	// TODO: Change this when status is available in cluster-operator
	time.Sleep(10 * time.Second)

	return nil
}

func installCalico(n nodes.Node, k string, privateParams PrivateParams, isNetPolEngine bool, dryRun bool) error {
	var c string
	var cmd exec.Cmd
	var err error
	keosCluster := privateParams.KeosCluster

	calicoTemplate := "/kind/tigera-operator-helm-values.yaml"

	calicoHelmParams := calicoHelmParams{
		Spec:           keosCluster.Spec,
		KeosRegUrl:     commons.GetPrefixedRegistryURL("docker.io", privateParams.KeosRegUrl, privateParams.CentralECR),
		QuayRegUrl:     commons.GetPrefixedRegistryURL("quay.io", privateParams.KeosRegUrl, privateParams.CentralECR),
		Private:        privateParams.Private,
		IsNetPolEngine: isNetPolEngine,
		Annotations: map[string]string{
			postInstallAnnotation: "var-lib-calico",
		},
	}

	// if CentralECR is enabled add suffix to the image
	if privateParams.CentralECR {
		privateParams.KeosRegUrl = commons.GetPrefixedRegistryURL("quay.io", privateParams.KeosRegUrl, privateParams.CentralECR)
	}
	// Generate the calico helm values
	calicoHelmValues, err := getManifest("common", "tigera-operator-helm-values.tmpl", majorVersion, calicoHelmParams)

	if err != nil {
		return errors.Wrap(err, "failed to generate calico helm values")
	}

	c = "echo '" + calicoHelmValues + "' > " + calicoTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create Calico Helm chart values file")
	}

	if !dryRun {
		c = "helm install tigera-operator /stratio/helm/tigera-operator" +
			" --kubeconfig " + k +
			" --namespace tigera-operator" +
			" --create-namespace" +
			" --values " + calicoTemplate
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to deploy Calico Helm Chart")
		}

		// Wait for calico-system namespace to be created
		c = "kubectl --kubeconfig " + kubeconfigPath + " get ns tigera-operator"
		_, err = commons.ExecuteCommand(n, c, 20, 5)
		if err != nil {
			return errors.Wrap(err, "failed to wait for tigera-operator namespace")
		}

		// Create calico metrics services
		cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-n", "tigera-operator", "-f", "-")
		if err = cmd.SetStdin(strings.NewReader(calicoMetrics)).Run(); err != nil {
			return errors.Wrap(err, "failed to create calico metrics services")
		}
	}
	return nil
}

func deployClusterAutoscaler(n nodes.Node, chartsList map[string]commons.ChartEntry, privateParams PrivateParams, capiClustersNamespace string, moveManagement bool) error {
	helmValuesCAFile := "/kind/cluster-autoscaler-helm-values.yaml"
	clusterAutoscalerEntry := chartsList["cluster-autoscaler"]
	clusterAutoscalerHelmReleaseParams := fluxHelmReleaseParams{
		HelmReleaseName: "cluster-autoscaler",
		ChartRepoRef:    "keos",
		ChartName:       "cluster-autoscaler",
		ChartNamespace:  clusterAutoscalerEntry.Namespace,
		ChartVersion:    clusterAutoscalerEntry.Version,
	}
	if !privateParams.HelmPrivate {
		clusterAutoscalerHelmReleaseParams.ChartRepoRef = "cluster-autoscaler"
	}

	// if Central ECR is enabled add suffix to the image
	if privateParams.CentralECR {
		privateParams.KeosRegUrl = commons.GetPrefixedRegistryURL("registry.k8s.io", privateParams.KeosRegUrl, privateParams.CentralECR)
	}
	helmValuesCA, err := getManifest("common", "cluster-autoscaler-helm-values.tmpl", majorVersion, privateParams)
	if err != nil {
		return errors.Wrap(err, "failed to get CA helm values")
	}
	c := "echo '" + helmValuesCA + "' > " + helmValuesCAFile
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create CA helm values file")
	}
	// Create Helm release using the fluxHelmReleaseParams
	if err := configureHelmRelease(n, kubeconfigPath, "flux2_helmrelease.tmpl", clusterAutoscalerHelmReleaseParams, privateParams.KeosCluster.Spec.HelmRepository); err != nil {
		return err
	}
	if !moveManagement {
		autoscalerRBACPath := "/kind/autoscaler_rbac.yaml"

		autoscalerRBAC, err := getManifest("common", "autoscaler_rbac.tmpl", "", privateParams.KeosCluster)
		if err != nil {
			return errors.Wrap(err, "failed to get CA RBAC file")
		}

		c = "echo '" + autoscalerRBAC + "' > " + autoscalerRBACPath
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create CA RBAC file")
		}

		// Create namespace for CAPI clusters (it must exists) in worker cluster
		c = "kubectl --kubeconfig " + kubeconfigPath + " create ns " + capiClustersNamespace
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create manifests Namespace")
		}

		c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + autoscalerRBACPath
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to apply CA RBAC")
		}
	}
	return nil
}

func configureFlux(n nodes.Node, k string, privateParams PrivateParams, helmRepoCreds HelmRegistry, keosClusterSpec commons.KeosSpec, chartsList map[string]commons.ChartEntry) error {
	var c string
	var err error

	fluxTemplate := "/kind/flux-helm-values.yaml"
	keosChartRepoScheme := "default"
	chartRepoScheme := "default"

	fluxHelmParams := commonHelmParams{
		KeosRegUrl: privateParams.KeosRegUrl,
		Private:    privateParams.Private,
	}

	// if Central ECR is enabled add suffix to the image
	if privateParams.CentralECR {
		fluxHelmParams.KeosRegUrl = commons.GetPrefixedRegistryURL("ghcr.io", privateParams.KeosRegUrl, privateParams.CentralECR)
	}

	// Generate the flux helm values
	fluxHelmValues, err := getManifest("common", "flux2-helm-values.tmpl", majorVersion, fluxHelmParams)

	if err != nil {
		return errors.Wrap(err, "failed to generate flux helm values")
	}

	c = "echo '" + fluxHelmValues + "' > " + fluxTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create Flux Helm chart values file")
	}

	c = "helm install flux /stratio/helm/flux2" +
		" --kubeconfig " + k +
		" --namespace kube-system" +
		" --create-namespace" +
		" --values " + fluxTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Flux Helm Chart")
	}

	// Set repository scheme for private case
	if strings.HasPrefix(helmRepoCreds.URL, "oci://") {
		keosChartRepoScheme = "oci"
	}

	var helmRepositoryInterval = "10m"

	// Create fluxHelmRepositoryParams for the private case
	fluxHelmRepositoryParams := fluxHelmRepositoryParams{
		ChartName:          "keos",
		ChartRepoUrl:       helmRepoCreds.URL,
		ChartRepoScheme:    keosChartRepoScheme,
		Spec:               keosClusterSpec,
		HelmRepoCreds:      helmRepoCreds,
		RepositoryInterval: helmRepositoryInterval,
	}

	if fluxHelmRepositoryParams.ChartName == "keos" && keosClusterSpec.HelmRepository.RepositoryInterval != "" {
		fluxHelmRepositoryParams.RepositoryInterval = keosClusterSpec.HelmRepository.RepositoryInterval
	}

	// Create Helm repository using the fluxHelmRepositoryParams
	if err := configureHelmRepository(n, k, "flux2_helmrepository.tmpl", fluxHelmRepositoryParams); err != nil {
		return err
	}

	// Update fluxHelmRepositoryParams if not private
	if !privateParams.HelmPrivate {
		// Iterate through charts and create Helm repositories and releases
		for name, entry := range chartsList {
			if entry.Repository != "default" {
				// Set repository scheme if it's oci
				if strings.HasPrefix(entry.Repository, "oci://") {
					chartRepoScheme = "oci"
				}

				// Update fluxHelmRepositoryParams if not private
				fluxHelmRepositoryParams.ChartName = name
				fluxHelmRepositoryParams.ChartRepoScheme = chartRepoScheme
				fluxHelmRepositoryParams.ChartRepoUrl = entry.Repository

				// Create Helm repository using the fluxHelmRepositoryParams
				if err := configureHelmRepository(n, k, "flux2_helmrepository.tmpl", fluxHelmRepositoryParams); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func reconcileCharts(n nodes.Node, k string, privateParams PrivateParams, keosClusterSpec commons.KeosSpec, chartsList map[string]commons.ChartEntry) error {
	// Iterate through charts and create Helm repositories and releases
	for name, entry := range chartsList {
		// Create fluxHelmReleaseParams for the current entry
		fluxHelmReleaseParams := fluxHelmReleaseParams{
			ChartRepoRef: "keos",
		}
		// Update fluxHelmRepositoryParams if not private
		if !privateParams.HelmPrivate && entry.Repository != "default" {
			fluxHelmReleaseParams.ChartRepoRef = name
		}

		// Adopt helm charts already deployed: tigera-operator and cloud-provider
		if entry.Reconcile {
			helmReleaseName := name
			if name == "flux2" {
				helmReleaseName = "flux"
			}
			fluxHelmReleaseParams.HelmReleaseName = helmReleaseName
			fluxHelmReleaseParams.ChartName = name
			fluxHelmReleaseParams.ChartNamespace = entry.Namespace
			fluxHelmReleaseParams.ChartVersion = entry.Version
			// tigera-operator-helm-values.yaml is required to install Calico as Network Policy engine

			if err := configureHelmRelease(n, k, "flux2_helmrelease.tmpl", fluxHelmReleaseParams, keosClusterSpec.HelmRepository); err != nil {
				return err
			}
		}
	}
	return nil
}

func configureHelmRepository(n nodes.Node, k string, templatePath string, params fluxHelmRepositoryParams) error {
	// Generate HelmRepository manifest
	fluxHelmRepository, err := getManifest("common", templatePath, majorVersion, params)
	if err != nil {
		return errors.Wrap(err, "failed to generate "+params.ChartName+" HelmRepository")
	}

	// Write HelmRepository manifest to file
	fluxHelmRepositoryTemplate := "/kind/" + params.ChartName + "_helmrepository.yaml"
	c := "echo '" + fluxHelmRepository + "' > " + fluxHelmRepositoryTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create "+params.ChartName+" Flux HelmRepository file")
	}

	// Apply HelmRepository
	c = "kubectl --kubeconfig " + k + " apply -f " + fluxHelmRepositoryTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy "+params.ChartName+" Flux HelmRepository")
	}
	return nil
}

func configureHelmRelease(n nodes.Node, k string, templatePath string, params fluxHelmReleaseParams, helmRepository commons.HelmRepository) error {
	valuesFile := "/kind/" + params.HelmReleaseName + "-helm-values.yaml"

	// Create default HelmRelease configmap
	c := "kubectl --kubeconfig " + kubeconfigPath + " " +
		"-n " + params.ChartNamespace + " create configmap " +
		"00-" + params.HelmReleaseName + "-helm-chart-default-values " +
		"--from-file=values.yaml=" + valuesFile
	_, err := commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy "+params.HelmReleaseName+" HelmRelease default configuration map")
	}

	// Create override HelmRelease configmap
	c = "kubectl --kubeconfig " + kubeconfigPath + " " +
		"-n " + params.ChartNamespace + " create configmap " +
		"02-" + params.HelmReleaseName + "-helm-chart-override-values " +
		"--from-literal=values.yaml=\"\""
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy "+params.HelmReleaseName+" HelmRelease override configmap")
	}

	var defaultHelmReleaseInterval = "1m"
	var defaultHelmReleaseRetries = 3
	var defaultHelmReleaseSourceInterval = "1m"

	completedfluxHelmReleaseParams := struct {
		HelmReleaseName           string
		ChartName                 string
		ChartNamespace            string
		ChartRepoRef              string
		ChartVersion              string
		HelmReleaseInterval       string
		HelmReleaseRetries        int
		HelmReleaseSourceInterval string
	}{
		HelmReleaseName:           params.HelmReleaseName,
		ChartName:                 params.ChartName,
		ChartNamespace:            params.ChartNamespace,
		ChartRepoRef:              params.ChartRepoRef,
		ChartVersion:              params.ChartVersion,
		HelmReleaseInterval:       defaultHelmReleaseInterval,
		HelmReleaseRetries:        defaultHelmReleaseRetries,
		HelmReleaseSourceInterval: defaultHelmReleaseSourceInterval,
	}

	if completedfluxHelmReleaseParams.ChartRepoRef == "keos" {
		if helmRepository.ReleaseInterval != "" {
			completedfluxHelmReleaseParams.HelmReleaseInterval = helmRepository.ReleaseInterval
		}
		if helmRepository.ReleaseSourceInterval != "" {
			completedfluxHelmReleaseParams.HelmReleaseSourceInterval = helmRepository.ReleaseSourceInterval
		}
		if helmRepository.ReleaseRetries != nil {
			completedfluxHelmReleaseParams.HelmReleaseRetries = *helmRepository.ReleaseRetries
		}
	}

	// Generate HelmRelease manifest
	fluxHelmHelmRelease, err := getManifest("common", templatePath, majorVersion, completedfluxHelmReleaseParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate "+params.HelmReleaseName+" HelmHelmRelease")
	}

	// Write HelmHelmRelease manifest to file
	fluxHelmHelmReleaseTemplate := "/kind/" + params.HelmReleaseName + "_helmrelease.yaml"
	c = "echo '" + fluxHelmHelmRelease + "' > " + fluxHelmHelmReleaseTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create "+params.HelmReleaseName+" Flux HelmHelmRelease file")
	}

	// Apply HelmHelmRelease
	c = "kubectl --kubeconfig " + k + " apply -f " + fluxHelmHelmReleaseTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to deploy "+params.HelmReleaseName+" Flux HelmHelmRelease")
	}

	// Check that the HelmRelease resource exists (a short retry is handled by ExecuteCommand)
	c = "kubectl --kubeconfig " + kubeconfigPath + " " +
		"-n " + params.ChartNamespace + " get helmrelease/" + params.HelmReleaseName
	_, err = commons.ExecuteCommand(n, c, 1, 2)
	if err != nil {
		return errors.Wrap(err, "failed to wait for "+params.HelmReleaseName+" HelmRelease to be created")
	}

	// Wait for HelmRelease to become ready
	c = "kubectl --kubeconfig " + kubeconfigPath + " " +
		"-n " + params.ChartNamespace + " wait helmrelease/" + params.HelmReleaseName +
		" --for=condition=ready --timeout=5m"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to wait for "+params.HelmReleaseName+" HelmRelease to become ready")
	}
	return nil
}

func customCoreDNS(n nodes.Node, keosCluster commons.KeosCluster) error {
	var c string
	var err error

	coreDNSPatchFile := "coredns"
	coreDNSTemplate := "/kind/coredns-configmap.yaml"
	coreDNSSuffix := ""

	if keosCluster.Spec.InfraProvider == "azure" && keosCluster.Spec.ControlPlane.Managed {
		coreDNSPatchFile = "coredns-custom"
		coreDNSSuffix = "-aks"
	}

	coreDNSConfigmap, err := getManifest(keosCluster.Spec.InfraProvider, "coredns_configmap"+coreDNSSuffix+".tmpl", "", keosCluster.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get CoreDNS file")
	}

	c = "echo '" + coreDNSConfigmap + "' > " + coreDNSTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create CoreDNS configmap file")
	}

	// Patch configmap
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system patch cm " + coreDNSPatchFile + " --patch-file " + coreDNSTemplate
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to customize coreDNS patching ConfigMap")
	}

	// Rollout restart to catch the made changes
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system rollout restart deploy coredns"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to redeploy coreDNS")
	}

	// Wait until CoreDNS completely rollout
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system rollout status deploy coredns --timeout=3m"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to wait for the customatization of CoreDNS configmap")
	}

	return nil
}

// installCAPXWorker installs CAPX in the worker cluster
func (p *Provider) installCAPXWorker(n nodes.Node, keosCluster commons.KeosCluster, clusterConfig commons.ClusterConfig, kubeconfigPath string) error {
	var c string
	var err error

	capxPDBPath := "/kind/capi_pdb.yaml"

	if p.capxProvider == "azure" {
		// Create capx namespace
		c = "kubectl --kubeconfig " + kubeconfigPath + " create namespace " + p.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		namespace := p.capxName + "-system"
		clientSecret, _ := base64.StdEncoding.DecodeString(strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET_B64=")[1])

		c := fmt.Sprintf(
			"kubectl --kubeconfig %s -n %s create secret generic cluster-identity-secret --from-literal=clientSecret='%s'",
			kubeconfigPath, namespace, clientSecret)
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	// Install CAPX in worker cluster
	c = "clusterctl --kubeconfig " + kubeconfigPath + " init --wait-providers" +
		" --core " + CAPICoreProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --bootstrap " + CAPIBootstrapProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --control-plane " + CAPIControlPlaneProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	_, err = commons.ExecuteCommand(n, c, 5, 3, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in workload cluster")
	}

	// GKE by default limits the consumption of this priority class using ResourceQuota
	if p.capxProvider == "gcp" && p.capxManaged {
		resourceQuotaPath := "/kind/resourceQuota.yaml"
		deploys := []struct {
			Name      string
			Namespace string
		}{{"capi", "capi-system"}, {p.capxName, p.capxName + "-system"}}
		for _, d := range deploys {
			resourceQuota, err := getManifest("gcp", "resourcequota.tmpl", majorVersion, d)
			if err != nil {
				return errors.Wrap(err, "failed to get ResourceQuota template")
			}
			c = "echo '" + resourceQuota + "' > " + resourceQuotaPath
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to save ResourceQuota manifest")
			}
			c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + resourceQuotaPath
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to apply ResourceQuota manifest")
			}
		}
	}

	// Manually assign PriorityClass to capx service
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system get deploy " + p.capxName + "-controller-manager -o jsonpath='{.spec.template.spec.priorityClassName}'"
	priorityClassName, err := commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to get priorityClass for "+p.capxName+"-controller-manager")
	}

	if priorityClassName != "system-node-critical" {
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system patch deploy " + p.capxName + "-controller-manager -p '{\"spec\": {\"template\": {\"spec\": {\"priorityClassName\": \"system-node-critical\"}}}}' --type=merge"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to assigned priorityClass to "+p.capxName+"-controller-manager")
		}
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system rollout status deploy " + p.capxName + "-controller-manager --timeout 60s"
		_, err = commons.ExecuteCommand(n, c, 30, 3)
		if err != nil {
			return errors.Wrap(err, "failed to check rollout status for "+p.capxName+"-controller-manager")
		}
	}

	// Scale CAPX to 2 replicas
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system scale --replicas 2 deploy " + p.capxName + "-controller-manager"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to scale CAPX in workload cluster")
	}
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system rollout status deploy " + p.capxName + "-controller-manager --timeout 60s"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to check rollout status for "+p.capxName+"-controller-manager")
	}

	// Define PodDisruptionBudget for capx services
	capxPDB, err := getManifest("common", "capx_pdb.tmpl", "", keosCluster.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get PodDisruptionBudget file")
	}

	c = "echo '" + capxPDB + "' > " + capxPDBPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create PodDisruptionBudget file")
	}

	c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + capxPDBPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to apply "+p.capxName+" PodDisruptionBudget")
	}

	return nil
}

func (p *Provider) configCAPIWorker(n nodes.Node, keosCluster commons.KeosCluster, kubeconfigPath string) error {
	var c string
	var err error
	var capiKubeadmReplicas int

	capiDeployments := []struct {
		name      string
		namespace string
	}{
		{name: "capi-controller-manager", namespace: "capi-system"},
		{name: "capi-kubeadm-control-plane-controller-manager", namespace: "capi-kubeadm-control-plane-system"},
		{name: "capi-kubeadm-bootstrap-controller-manager", namespace: "capi-kubeadm-bootstrap-system"},
	}

	allowedNamePattern := regexp.MustCompile(`^capi-kubeadm-(control-plane|bootstrap)-controller-manager$`)
	capiPDBPath := "/kind/capi_pdb.yaml"

	// Determine the number of replicas for capi-kubeadm deployments
	if p.capxManaged {
		capiKubeadmReplicas = 0
	} else {
		capiKubeadmReplicas = 2
	}

	// Manually assign PriorityClass to capi services
	for _, deployment := range capiDeployments {
		if !p.capxManaged || (p.capxManaged && !allowedNamePattern.MatchString(deployment.name)) {
			c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + deployment.namespace + " patch deploy " + deployment.name + " -p '{\"spec\": {\"template\": {\"spec\": {\"priorityClassName\": \"system-node-critical\"}}}}' --type=merge"
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to assigned priorityClass to "+deployment.name)
			}
		}
	}

	// Scale number of replicas to 2 for capi service
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n capi-system scale deploy capi-controller-manager --replicas 2"
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to scale the CAPI Deployment")
	}
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to check rollout status for capi-controller-manager")
	}

	// Scale number of required replicas for capi kubeadm services
	for _, deployment := range capiDeployments {
		if deployment.name != "capi-controller-manager" {
			c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + deployment.namespace + " scale --replicas " + strconv.Itoa(capiKubeadmReplicas) + " deploy " + deployment.name
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to scale the "+deployment.name+" deployment")
			}
			c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + deployment.namespace + " rollout status deploy " + deployment.name + " --timeout 60s"
			_, err = commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to check rollout status for "+deployment.name)
			}
		}
	}

	// Define PodDisruptionBudget for capi services
	capiPDB, err := getManifest("common", "capi_pdb.tmpl", "", keosCluster.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get PodDisruptionBudget file")
	}
	c = "echo '" + capiPDB + "' > " + capiPDBPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create PodDisruptionBudget file")
	}

	c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + capiPDBPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)

	if err != nil {
		return errors.Wrap(err, "failed to apply "+p.capxName+" PodDisruptionBudget")
	}

	return nil
}

// installCAPXLocal installs CAPX in the local cluster
func (p *Provider) installCAPXLocal(n nodes.Node, clusterConfig commons.ClusterConfig, providerParams ProviderParams) error {
	var c string
	var err error

	if p.capxProvider == "azure" {
		// Create capx namespace
		c = "kubectl create namespace " + p.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		namespace := p.capxName + "-system"
		clientSecret, _ := base64.StdEncoding.DecodeString(strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET_B64=")[1])

		c := fmt.Sprintf(
			"kubectl -n %s create secret generic cluster-identity-secret "+
				"--from-literal=clientSecret='%s' ",
			namespace, clientSecret)
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	c = "clusterctl init --wait-providers" +
		" --core " + CAPICoreProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --bootstrap " + CAPIBootstrapProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --control-plane " + CAPIControlPlaneProvider + ":" + clusterConfig.Spec.Capx.CAPI_Version +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	_, err = commons.ExecuteCommand(n, c, 5, 3, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in local cluster")
	}

	// [EKS] If we are using assume role, update capa-manager-bootstrap-credentials secret
	if p.capxProvider == "aws" && providerParams.Credentials["RoleARN"] != "" {
		// Update secret capa-manager-bootstrap-credentials with new credentials
		providerSecrets := providerParams.Credentials
		var cfg aws.Config
		var err error
		// Step 1: Get AWS Config in order to retrieve new credentials with session token
		cfg, err = commons.AWSGetConfig(context.TODO(), providerSecrets)
		if err != nil {
			return err
		}
		// Step 2: Retrieve new credentials
		creds, err := cfg.Credentials.Retrieve(context.TODO())
		if err != nil {
			return errors.Wrap(err, "failed to retrieve new credentials")
		}
		// Step 3: Encode new credentials to base64
		credentialsString := fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s\naws_session_token=%s\n", creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
		credentialsString = base64.StdEncoding.EncodeToString([]byte(credentialsString))
		// Step 4: Patch secret with new credentials
		c = "kubectl -n capa-system patch secret capa-manager-bootstrap-credentials -p '{\"data\":{\"credentials\":\"" + credentialsString + "\"}}'"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to update capa-manager-bootstrap-credentials secret")
		}
		// Step 5: Rollout restart capa-controller-manager
		c = "kubectl -n capa-system rollout restart deployment capa-controller-manager"
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to restart capa-controller-manager")
		}
	}

	return nil
}

func enableSelfHealing(n nodes.Node, keosCluster commons.KeosCluster, namespace string, clusterConfig *commons.ClusterConfig) error {
	var c string
	var err error

	if !keosCluster.Spec.ControlPlane.Managed {
		machineRole := "-control-plane-node"
		controlplane_maxunhealty := 34
		if clusterConfig != nil {
			if clusterConfig.Spec.ControlplaneConfig.MaxUnhealthy != nil {
				controlplane_maxunhealty = *clusterConfig.Spec.ControlplaneConfig.MaxUnhealthy
			}
		}

		err = generateMHCManifest(n, keosCluster.Metadata.Name, namespace, machineHealthCheckControlPlaneNodePath, machineRole, controlplane_maxunhealty)
		if err != nil {
			return errors.Wrap(err, "failed to create the MachineHealthCheck manifest")
		}
		c = "kubectl -n " + namespace + " apply -f " + machineHealthCheckControlPlaneNodePath
		_, err = commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
		}
	}

	machineRole := "-worker-node"
	workernode_maxunhealty := 34
	if clusterConfig != nil {
		if clusterConfig.Spec.WorkersConfig.MaxUnhealthy != nil {
			workernode_maxunhealty = *clusterConfig.Spec.WorkersConfig.MaxUnhealthy
		}
	}
	err = generateMHCManifest(n, keosCluster.Metadata.Name, namespace, machineHealthCheckWorkerNodePath, machineRole, workernode_maxunhealty)
	if err != nil {
		return errors.Wrap(err, "failed to create the MachineHealthCheck manifest")
	}
	c = "kubectl -n " + namespace + " apply -f " + machineHealthCheckWorkerNodePath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
	}

	return nil
}

func generateMHCManifest(n nodes.Node, clusterID string, namespace string, manifestPath string, machineRole string, maxunhealthy int) error {
	var c string
	var err error
	var maxUnhealthy = strconv.Itoa(maxunhealthy) + "%"

	var machineHealthCheck = `
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
metadata:
  name: ` + clusterID + machineRole + `-unhealthy
  namespace: ` + namespace + `
spec:
  clusterName: ` + clusterID + `
  nodeStartupTimeout: 300s
  maxUnhealthy: ` + maxUnhealthy + `
  selector:
    matchLabels:
      keos.stratio.com/machine-role: ` + clusterID + machineRole + `
  unhealthyConditions:
    - type: Ready
      status: Unknown
      timeout: 180s
    - type: Ready
      status: 'False'
      timeout: 180s`

	c = "echo \"" + machineHealthCheck + "\" > " + manifestPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to write the MachineHealthCheck manifest")
	}

	return nil
}

func getManifest(parentPath string, name string, majorVersion string, params interface{}) (string, error) {
	templatePath := filepath.Join("templates", parentPath, majorVersion, name)
	if majorVersion == "" {
		templatePath = filepath.Join("templates", parentPath, name)
	}

	var tpl bytes.Buffer
	t, err := template.New("").ParseFS(ctel, templatePath)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, name, params)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func patchDeploy(n nodes.Node, k string, ns string, deployName string, patch string) error {
	c := "kubectl --kubeconfig " + k + " patch deploy -n " + ns + " " + deployName + " -p '" + patch + "'"
	_, err := commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return err
	}
	return rolloutStatus(n, k, ns, deployName)
}

func rolloutStatus(n nodes.Node, k string, ns string, deployName string) error {
	c := "kubectl --kubeconfig " + k + " rollout status deploy -n " + ns + " " + deployName + " --timeout=5m"
	_, err := commons.ExecuteCommand(n, c, 5, 3)
	return err
}

func installCorednsPdb(n nodes.Node) error {

	// Define PodDisruptionBudget for coredns service
	corednsPDBLocalPath := "files/common/coredns_pdb.yaml"
	corednsPDB, err := getcapxPDB(corednsPDBLocalPath)
	if err != nil {
		return err
	}

	c := "echo \"" + corednsPDB + "\" > " + corednsPdbPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create coredns PodDisruptionBudget file")
	}

	c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + corednsPdbPath
	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to apply coredns PodDisruptionBudget")
	}
	return nil
}

func pullCharts(n nodes.Node, charts map[string]commons.ChartEntry, keosSpec commons.KeosSpec, clusterCredentials commons.ClusterCredentials) error {
	for name, chart := range charts {
		// Set default repository if needed
		if chart.Repository == "default" {
			chart.Repository = keosSpec.HelmRepository.URL
		}
		// Check if the chart needs to be pulled
		if chart.Pull {
			var c string
			if strings.HasPrefix(chart.Repository, "oci://") {
				c = "helm pull " + chart.Repository + "/" + name + " --version " + chart.Version + " --untar --untardir /stratio/helm"
			} else {
				c = "helm pull " + name + " --version " + chart.Version + " --repo " + chart.Repository + " --untar --untardir /stratio/helm"
			}
			// Add authentication if required
			if chart.Repository == keosSpec.HelmRepository.URL && keosSpec.HelmRepository.AuthRequired {
				if keosSpec.HelmRepository.AuthRequired {
					c = c + " --username " + clusterCredentials.HelmRepositoryCredentials["User"] + " --password " + clusterCredentials.HelmRepositoryCredentials["Pass"]
				}
			}
			// Execute the command
			_, err := commons.ExecuteCommand(n, c, 5, 3)
			if err != nil {
				return errors.Wrap(err, "failed to pull the helm chart: "+fmt.Sprint(chart))
			}
		}
	}
	return nil
}

func loginHelmRepo(n nodes.Node, keosCluster commons.KeosCluster, clusterCredentials commons.ClusterCredentials, helmRepoCreds *HelmRegistry, infra *Infra, providerParams ProviderParams) error {

	var helmRepository helmRepository
	var err error

	helmRepoCreds.Type = keosCluster.Spec.HelmRepository.Type
	helmRepoCreds.URL = keosCluster.Spec.HelmRepository.URL
	if keosCluster.Spec.HelmRepository.Type != "generic" {
		urlLogin := strings.Split(strings.Split(helmRepoCreds.URL, "//")[1], "/")[0]
		helmRepoCreds.User, helmRepoCreds.Pass, err = infra.getRegistryCredentials(providerParams, urlLogin)
		if err != nil {
			return errors.Wrap(err, "failed to get helm registry credentials")
		}
	} else {
		helmRepoCreds.User = clusterCredentials.HelmRepositoryCredentials["User"]
		helmRepoCreds.Pass = clusterCredentials.HelmRepositoryCredentials["Pass"]
	}

	if strings.HasPrefix(keosCluster.Spec.HelmRepository.URL, "oci://") {
		stratio_helm_repo = helmRepoCreds.URL
		urlLogin := strings.Split(strings.Split(keosCluster.Spec.HelmRepository.URL, "//")[1], "/")[0]

		c := "helm registry login " + urlLogin + " --username " + helmRepoCreds.User + " --password " + helmRepoCreds.Pass
		_, err := commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to add and authenticate to helm repository: "+helmRepoCreds.URL)
		}
	} else if keosCluster.Spec.HelmRepository.AuthRequired {
		helmRepository.user = clusterCredentials.HelmRepositoryCredentials["User"]
		helmRepository.pass = clusterCredentials.HelmRepositoryCredentials["Pass"]
		stratio_helm_repo = "stratio-helm-repo"
		c := "helm repo add " + stratio_helm_repo + " " + helmRepoCreds.URL + " --username " + helmRepoCreds.User + " --password " + helmRepoCreds.Pass
		_, err := commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to add and authenticate to helm repository: "+helmRepository.url)
		}
	} else {
		stratio_helm_repo = "stratio-helm-repo"
		c := "helm repo add " + stratio_helm_repo + " " + helmRepoCreds.URL
		_, err := commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to add helm repository: "+helmRepoCreds.URL)
		}
	}
	return nil
}

func getChartVersion(charts []commons.Chart, chartName string) string {
	for _, chart := range charts {
		if chart.Name == chartName {
			return chart.Version
		}
	}
	return ""
}
