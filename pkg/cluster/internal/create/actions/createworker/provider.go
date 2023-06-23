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
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed templates/*
var ctel embed.FS

const (
	CAPICoreProvider         = "cluster-api:v1.4.3"
	CAPIBootstrapProvider    = "kubeadm:v1.4.3"
	CAPIControlPlaneProvider = "kubeadm:v1.4.3"
)

const machineHealthCheckWorkerNodePath = "/kind/manifests/machinehealthcheckworkernode.yaml"
const machineHealthCheckControlPlaneNodePath = "/kind/manifests/machinehealthcheckcontrolplane.yaml"

//go:embed files/calico-metrics.yaml
var calicoMetrics string

type PBuilder interface {
	setCapx(managed bool)
	setCapxEnvVars(p commons.ProviderParams)
	installCSI(n nodes.Node, k string) error
	getProvider() Provider
	configureStorageClass(n nodes.Node, k string, sc commons.StorageClass) error
	getParameters(sc commons.StorageClass) commons.SCParameters
	getAzs(networks commons.Networks) ([]string, error)
	internalNginx(networks commons.Networks, credentialsMap map[string]string, clusterID string) (bool, error)
}

type Provider struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxName         string
	capxTemplate     string
	capxEnvVars      []string
	stClassName      string
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

type StorageClassDef struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Name        string            `yaml:"name"`
	} `yaml:"metadata"`
	Provisioner       string                 `yaml:"provisioner"`
	Parameters        map[string]interface{} `yaml:"parameters"`
	VolumeBindingMode string                 `yaml:"volumeBindingMode"`
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

func (i *Infra) buildProvider(p commons.ProviderParams) Provider {
	i.builder.setCapx(p.Managed)
	i.builder.setCapxEnvVars(p)
	return i.builder.getProvider()
}

func (i *Infra) installCSI(n nodes.Node, k string) error {
	return i.builder.installCSI(n, k)
}

func (i *Infra) configureStorageClass(n nodes.Node, k string, sc commons.StorageClass) error {
	return i.builder.configureStorageClass(n, k, sc)
}

func (i *Infra) internalNginx(networks commons.Networks, credentialsMap map[string]string, ClusterID string) (bool, error) {
	requiredIntenalNginx, err := i.builder.internalNginx(networks, credentialsMap, ClusterID)
	if err != nil {
		return false, err
	}
	return requiredIntenalNginx, nil
}

func (i *Infra) getAzs(networks commons.Networks) ([]string, error) {
	azs, err := i.builder.getAzs(networks)
	if err != nil {
		return nil, err
	}
	return azs, nil
}

func installCalico(n nodes.Node, k string, descriptorFile commons.DescriptorFile, allowCommonEgressNetPolPath string) error {
	var c string
	var cmd exec.Cmd
	var err error

	calicoTemplate := "/kind/calico-helm-values.yaml"

	// Generate the calico helm values
	calicoHelmValues, err := getManifest("calico-helm-values.tmpl", descriptorFile)
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
	c = "timeout 30s bash -c 'until kubectl --kubeconfig " + kubeconfigPath + " get ns calico-system; do sleep 2s ; done'"
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

func enableSelfHealing(n nodes.Node, descriptorFile commons.DescriptorFile, namespace string) error {
	var c string
	var err error

	if !descriptorFile.ControlPlane.Managed {
		machineRole := "-control-plane-node"
		generateMHCManifest(n, descriptorFile.ClusterID, namespace, machineHealthCheckControlPlaneNodePath, machineRole)

		c = "kubectl -n " + namespace + " apply -f " + machineHealthCheckControlPlaneNodePath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
		}
	}

	machineRole := "-worker-node"
	generateMHCManifest(n, descriptorFile.ClusterID, namespace, machineHealthCheckWorkerNodePath, machineRole)

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
      timeout: 60s
    - type: Ready
      status: 'False'
      timeout: 60s`

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

func GetClusterManifest(flavor string, params commons.TemplateParams, azs []string) (string, error) {
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
		"inc": func(i int) int {
			return i + 1
		},
		"base64": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"sub":   func(a, b int) int { return a - b },
		"split": strings.Split,
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

func getManifest(name string, params interface{}) (string, error) {
	var tpl bytes.Buffer
	t, err := template.New("").ParseFS(ctel, "templates/"+name)
	if err != nil {
		return "", err
	}

	err = t.ExecuteTemplate(&tpl, name, params)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func setStorageClassParameters(storageClass string, params map[string]string, lineToStart string) (string, error) {

	paramIndex := strings.Index(storageClass, lineToStart)
	if paramIndex == -1 {
		return storageClass, nil
	}

	var lines []string
	for key, value := range params {
		line := "  " + key + ": " + value
		lines = append(lines, line)
	}

	linesToInsert := "\n" + strings.Join(lines, "\n") + "\n"
	newStorageClass := storageClass[:paramIndex+len(lineToStart)] + linesToInsert + storageClass[paramIndex+len(lineToStart):]

	return newStorageClass, nil
}

func mergeSCParameters(params1, params2 commons.SCParameters) commons.SCParameters {
	destValue := reflect.ValueOf(&params1).Elem()
	srcValue := reflect.ValueOf(&params2).Elem()

	for i := 0; i < srcValue.NumField(); i++ {
		srcField := srcValue.Field(i)
		destField := destValue.Field(i)

		if srcField.IsValid() && destField.IsValid() && destField.CanSet() {
			destFieldValue := destField.Interface()

			if reflect.DeepEqual(destFieldValue, reflect.Zero(destField.Type()).Interface()) {
				destField.Set(srcField)
			}
		}
	}

	return params1
}

func insertParameters(storageClass StorageClassDef, params commons.SCParameters) (string, error) {
	paramsYAML, err := structToYAML(params)
	if err != nil {
		return "", err
	}

	newMap := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(paramsYAML), &newMap)
	if err != nil {
		return "", err
	}

	for key, value := range newMap {
		newKey := strings.ReplaceAll(key, "_", "-")
		storageClass.Parameters[newKey] = value
	}

	if storageClass.Provisioner == "ebs.csi.aws.com" {
		if labels, ok := storageClass.Parameters["labels"].(string); ok && labels != "" {
			delete(storageClass.Parameters, "labels")
			for i, label := range strings.Split(labels, ",") {
				key_prefix := "tagSpecification_"
				key := key_prefix + strconv.Itoa(i)
				storageClass.Parameters[key] = label
			}
		}
	}

	resultYAML, err := yaml.Marshal(storageClass)
	if err != nil {
		return "", err
	}

	return string(resultYAML), nil
}

func structToYAML(data interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}
