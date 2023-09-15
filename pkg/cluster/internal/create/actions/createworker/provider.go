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
	"embed"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"time"

	"strings"
	"text/template"

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

const (
	CAPICoreProvider         = "cluster-api:v1.5.1"
	CAPIBootstrapProvider    = "kubeadm:v1.5.1"
	CAPIControlPlaneProvider = "kubeadm:v1.5.1"

	scName = "keos"

	keosClusterChart = "0.1.0"
	keosClusterImage = "0.1.0"
)

const machineHealthCheckWorkerNodePath = "/kind/manifests/machinehealthcheckworkernode.yaml"
const machineHealthCheckControlPlaneNodePath = "/kind/manifests/machinehealthcheckcontrolplane.yaml"
const defaultScAnnotation = "storageclass.kubernetes.io/is-default-class"

//go:embed files/common/calico-metrics.yaml
var calicoMetrics string

type PBuilder interface {
	setCapx(managed bool)
	setCapxEnvVars(p ProviderParams)
	setSC(p ProviderParams)
	installCloudProvider(n nodes.Node, k string, keosCluster commons.KeosCluster) error
	installCSI(n nodes.Node, k string) error
	getProvider() Provider
	configureStorageClass(n nodes.Node, k string) error
	getAzs(p ProviderParams, networks commons.Networks) ([]string, error)
	internalNginx(p ProviderParams, networks commons.Networks) (bool, error)
	getOverrideVars(p ProviderParams, networks commons.Networks) (map[string][]byte, error)
}

type Provider struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxManaged      bool
	capxName         string
	capxTemplate     string
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
	i.builder.setCapx(p.Managed)
	i.builder.setCapxEnvVars(p)
	i.builder.setSC(p)
	return i.builder.getProvider()
}

func (i *Infra) installCloudProvider(n nodes.Node, k string, keosCluster commons.KeosCluster) error {
	return i.builder.installCloudProvider(n, k, keosCluster)
}

func (i *Infra) installCSI(n nodes.Node, k string) error {
	return i.builder.installCSI(n, k)
}

func (i *Infra) configureStorageClass(n nodes.Node, k string) error {
	return i.builder.configureStorageClass(n, k)
}

func (i *Infra) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	return i.builder.internalNginx(p, networks)
}

func (i *Infra) getOverrideVars(p ProviderParams, networks commons.Networks) (map[string][]byte, error) {
	return i.builder.getOverrideVars(p, networks)
}

func (i *Infra) getAzs(p ProviderParams, networks commons.Networks) ([]string, error) {
	return i.builder.getAzs(p, networks)
}

func (p *Provider) getDenyAllEgressIMDSGNetPol() (string, error) {
	denyAllEgressIMDSGNetPolLocalPath := "files/" + p.capxProvider + "/deny-all-egress-imds_gnetpol.yaml"
	denyAllEgressIMDSgnpFile, err := denyAllEgressIMDSgnpFiles.Open(denyAllEgressIMDSGNetPolLocalPath)
	if err != nil {
		return "", errors.Wrap(err, "error opening the deny all egress IMDS file")
	}
	defer denyAllEgressIMDSgnpFile.Close()
	denyAllEgressIMDSgnpContent, err := ioutil.ReadAll(denyAllEgressIMDSgnpFile)
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
	allowEgressIMDSgnpContent, err := ioutil.ReadAll(allowEgressIMDSgnpFile)
	if err != nil {
		return "", err
	}

	return string(allowEgressIMDSgnpContent), nil
}

func deployClusterOperator(n nodes.Node, keosCluster commons.KeosCluster, clusterCredentials commons.ClusterCredentials, keosRegistry keosRegistry, kubeconfigPath string, firstInstallation bool) error {
	var c string
	var err error
	var helmRepository helmRepository

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
		if keosCluster.Spec.ControlPlane.Managed {
			keosCluster.Spec.ControlPlane.HighlyAvailable = nil
		}
		keosCluster.Spec.Keos = struct {
			Flavour string `yaml:"flavour,omitempty"`
			Version string `yaml:"version,omitempty"`
		}{}
		keosClusterYAML, err := yaml.Marshal(keosCluster)
		if err != nil {
			return err
		}
		// Write keoscluster file
		c = "echo '" + string(keosClusterYAML) + "' > " + manifestsPath + "/keoscluster.yaml"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to write the keoscluster file")
		}
		// Add helm repository
		helmRepository.url = keosCluster.Spec.HelmRepository.URL
		if keosCluster.Spec.HelmRepository.AuthRequired {
			helmRepository.user = clusterCredentials.HelmRepositoryCredentials["User"]
			helmRepository.pass = clusterCredentials.HelmRepositoryCredentials["Pass"]
			c = "helm repo add stratio-helm-repo " + helmRepository.url + " --username " + helmRepository.user + " --password " + helmRepository.pass
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to add and authenticate to helm repository: "+helmRepository.url)
			}
		} else {
			c = "helm repo add stratio-helm-repo " + helmRepository.url
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to add helm repository: "+helmRepository.url)
			}
		}
		if firstInstallation {
			// Pull cluster operator helm chart
			c = "helm pull cluster-operator --repo " + helmRepository.url +
				" --version " + keosClusterChart +
				" --untar --untardir /stratio/helm"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to pull cluster operator helm chart")
			}
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
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create keoscluster-registries secret")
		}
	}

	// Deploy keoscluster-controller-manager chart
	c = "helm install --wait cluster-operator /stratio/helm/cluster-operator" +
		" --namespace kube-system" +
		" --set app.containers.controllerManager.image.registry=" + keosRegistry.url +
		" --set app.containers.controllerManager.image.repository=stratio/cluster-operator" +
		" --set app.containers.controllerManager.image.tag=" + keosClusterImage
	if kubeconfigPath == "" {
		c = c +
			" --set app.containers.controllerManager.imagePullSecrets.enabled=true" +
			" --set app.containers.controllerManager.imagePullSecrets.name=regcred"
	} else {
		c = c + " --set app.replicas=2" + " --kubeconfig " + kubeconfigPath
	}
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy keoscluster-controller-manager chart")
	}

	// Wait for keoscluster-controller-manager deployment
	c = "kubectl -n kube-system rollout status deploy/keoscluster-controller-manager --timeout=3m"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to wait for keoscluster-controller-manager deployment")
	}

	// TODO: Change this when status is available in keoscluster-controller-manager
	time.Sleep(10 * time.Second)

	return nil
}

func installCalico(n nodes.Node, k string, keosCluster commons.KeosCluster, allowCommonEgressNetPolPath string) error {
	var c string
	var cmd exec.Cmd
	var err error

	calicoTemplate := "/kind/calico-helm-values.yaml"

	// Generate the calico helm values
	calicoHelmValues, err := getManifest("common", "calico-helm-values.tmpl", keosCluster.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to generate calico helm values")
	}

	c = "echo '" + calicoHelmValues + "' > " + calicoTemplate
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create Calico Helm chart values file")
	}

	c = "helm install calico /stratio/helm/tigera-operator" +
		" --kubeconfig " + k +
		" --namespace tigera-operator" +
		" --create-namespace" +
		" --values " + calicoTemplate
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Calico Helm Chart")
	}

	// Allow egress in tigera-operator namespace
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n tigera-operator apply -f " + allowCommonEgressNetPolPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply tigera-operator egress NetworkPolicy")
	}

	// Wait for calico-system namespace to be created
	c = "timeout 300s bash -c 'until kubectl --kubeconfig " + kubeconfigPath + " get ns calico-system; do sleep 2s ; done'"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to wait for calico-system namespace")
	}

	// Allow egress in calico-system namespace
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n calico-system apply -f " + allowCommonEgressNetPolPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply calico-system egress NetworkPolicy")
	}

	// Create calico metrics services
	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(calicoMetrics)).Run(); err != nil {
		return errors.Wrap(err, "failed to create calico metrics services")
	}

	return nil
}

func customCoreDNS(n nodes.Node, k string, keosCluster commons.KeosCluster) error {
	var c string
	var err error

	coreDNSPatchFile := "coredns"
	coreDNSTemplate := "/kind/coredns-configmap.yaml"
	coreDNSSuffix := ""

	if keosCluster.Spec.InfraProvider == "azure" && keosCluster.Spec.ControlPlane.Managed {
		coreDNSPatchFile = "coredns-custom"
		coreDNSSuffix = "-aks"
	}

	coreDNSConfigmap, err := getManifest(keosCluster.Spec.InfraProvider, "coredns_configmap"+coreDNSSuffix+".tmpl", keosCluster.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get CoreDNS file")
	}

	c = "echo '" + coreDNSConfigmap + "' > " + coreDNSTemplate
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create CoreDNS configmap file")
	}

	// Patch configmap
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system patch cm " + coreDNSPatchFile + " --patch-file " + coreDNSTemplate
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to customize coreDNS patching ConfigMap")
	}

	// Rollout restart to catch the made changes
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system rollout restart deploy coredns"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to redeploy coreDNS")
	}

	// Wait until CoreDNS completely rollout
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system rollout status deploy coredns --timeout=3m"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to wait for the customatization of CoreDNS configmap")
	}

	return nil
}

// installCAPXWorker installs CAPX in the worker cluster
func (p *Provider) installCAPXWorker(n nodes.Node, kubeconfigPath string, allowAllEgressNetPolPath string) error {
	var c string
	var err error

	if p.capxProvider == "azure" {
		// Create capx namespace
		c = "kubectl --kubeconfig " + kubeconfigPath + " create namespace " + p.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		secret := strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET=")[1]
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system create secret generic cluster-identity-secret --from-literal=clientSecret='" + string(secret) + "'"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	// Install CAPX in worker cluster
	c = "clusterctl --kubeconfig " + kubeconfigPath + " init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	_, err = commons.ExecuteCommand(n, c, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in workload cluster")
	}

	// Scale CAPX to 2 replicas
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system scale --replicas 2 deploy " + p.capxName + "-controller-manager"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to scale CAPX in workload cluster")
	}

	// Allow egress in CAPX's Namespace
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system apply -f " + allowAllEgressNetPolPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply CAPX's NetworkPolicy in workload cluster")
	}

	return nil
}

// installCAPXLocal installs CAPX in the local cluster
func (p *Provider) installCAPXLocal(n nodes.Node) error {
	var c string
	var err error

	if p.capxProvider == "azure" {
		// Create capx namespace
		c = "kubectl create namespace " + p.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		secret := strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET=")[1]
		c = "kubectl -n " + p.capxName + "-system create secret generic cluster-identity-secret --from-literal=clientSecret='" + string(secret) + "'"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	c = "clusterctl init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	_, err = commons.ExecuteCommand(n, c, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in local cluster")
	}

	return nil
}

func enableSelfHealing(n nodes.Node, keosCluster commons.KeosCluster, namespace string) error {
	var c string
	var err error

	if !keosCluster.Spec.ControlPlane.Managed {
		machineRole := "-control-plane-node"
		generateMHCManifest(n, keosCluster.Metadata.Name, namespace, machineHealthCheckControlPlaneNodePath, machineRole)

		c = "kubectl -n " + namespace + " apply -f " + machineHealthCheckControlPlaneNodePath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
		}
	}

	machineRole := "-worker-node"
	generateMHCManifest(n, keosCluster.Metadata.Name, namespace, machineHealthCheckWorkerNodePath, machineRole)

	c = "kubectl -n " + namespace + " apply -f " + machineHealthCheckWorkerNodePath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
	}

	return nil
}

func generateMHCManifest(n nodes.Node, clusterID string, namespace string, manifestPath string, machineRole string) error {
	var c string
	var err error
	var machineHealthCheck = `
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
metadata:
  name: ` + clusterID + machineRole + `-unhealthy
  namespace: cluster-` + clusterID + `
spec:
  clusterName: ` + clusterID + `
  nodeStartupTimeout: 300s
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
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to write the MachineHealthCheck manifest")
	}

	return nil
}

func resto(n int, i int, azs int) int {
	var r int
	r = (n % azs) / (i + 1)
	if r > 1 {
		r = 1
	}
	return r
}

func GetClusterManifest(params commons.TemplateParams) (string, error) {
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
					for i, a := range params.ProviderAZs {
						if zd == "unbalanced" {
							q = qa/len(params.ProviderAZs) + resto(qa, i, len(params.ProviderAZs))
							mx = maxsize/len(params.ProviderAZs) + resto(maxsize, i, len(params.ProviderAZs))
							mn = minsize/len(params.ProviderAZs) + resto(minsize, i, len(params.ProviderAZs))
							ch <- Node{AZ: a, QA: q, MaxSize: mx, MinSize: mn}
						} else {
							ch <- Node{AZ: a, QA: qa / len(params.ProviderAZs), MaxSize: maxsize / len(params.ProviderAZs), MinSize: minsize / len(params.ProviderAZs)}
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
		"inc": func(i int) int {
			return i + 1
		},
		"base64": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"sub":   func(a, b int) int { return a - b },
		"split": strings.Split,
	}
	templatePath := filepath.Join("templates", params.KeosCluster.Spec.InfraProvider, params.Flavor)

	var tpl bytes.Buffer
	t, err := template.New("").Funcs(funcMap).ParseFS(ctel, templatePath)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, params.Flavor, params)
	if err != nil {
		return "", err
	}

	return tpl.String(), nil
}

func getManifest(parentPath string, name string, params interface{}) (string, error) {
	templatePath := filepath.Join("templates", parentPath, name)

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
