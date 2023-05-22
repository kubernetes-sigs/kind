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
	"strings"

	"text/template"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

//go:embed templates/*
var ctel embed.FS

const (
	CAPICoreProvider         = "cluster-api:v1.4.1"
	CAPIBootstrapProvider    = "kubeadm:v1.4.1"
	CAPIControlPlaneProvider = "kubeadm:v1.4.1"
)

const machineHealthCheckWorkerNodePath = "/kind/manifests/machinehealthcheckworkernode.yaml"
const machineHealthCheckControlPlaneNodePath = "/kind/manifests/machinehealthcheckcontrolplane.yaml"

type PBuilder interface {
	setCapx(managed bool)
	setCapxEnvVars(p commons.ProviderParams)
	installCSI(n nodes.Node, k string) error
	getProvider() Provider
	getAzs() ([]string, error)
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

func getBuilder(builderType string) PBuilder {
	if builderType == "aws" {
		return newAWSBuilder()
	}

	if builderType == "gcp" {
		return newGCPBuilder()
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

func (i *Infra) getAzs() ([]string, error) {
	azs, err := i.builder.getAzs()
	if err != nil {
		return nil, err
	}
	return azs, nil
}

func installCalico(n nodes.Node, k string, descriptorFile commons.DescriptorFile, allowCommonEgressNetPolPath string) error {
	var c string
	var err error

	calicoTemplate := "/kind/calico-helm-values.yaml"

	// Generate the calico helm values
	calicoHelmValues, err := getManifest("calico-helm-values.tmpl", descriptorFile)
	if err != nil {
		return errors.Wrap(err, "failed to generate calico helm values")
	}

	c = "echo '" + calicoHelmValues + "' > " + calicoTemplate
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create Calico Helm chart values file")
	}

	c = "helm install calico /stratio/helm/tigera-operator" +
		" --kubeconfig " + k +
		" --namespace tigera-operator" +
		" --create-namespace" +
		" --values " + calicoTemplate
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Calico Helm Chart")
	}

	// Allow egress in tigera-operator namespace
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n tigera-operator apply -f " + allowCommonEgressNetPolPath
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply tigera-operator egress NetworkPolicy")
	}

	// Wait for calico-system namespace to be created
	c = "timeout 30s bash -c 'until kubectl --kubeconfig " + kubeconfigPath + " get ns calico-system; do sleep 2s ; done'"
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to wait for calico-system namespace")
	}

	// Allow egress in calico-system namespace
	c = "kubectl --kubeconfig " + kubeconfigPath + " -n calico-system apply -f " + allowCommonEgressNetPolPath
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to apply calico-system egress NetworkPolicy")
	}

	return nil
}

// installCAPXWorker installs CAPX in the worker cluster
func (p *Provider) installCAPXWorker(node nodes.Node, kubeconfigPath string, allowAllEgressNetPolPath string) error {
	var command string
	var err error

	// Install CAPX in worker cluster
	command = "clusterctl --kubeconfig " + kubeconfigPath + " init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	err = commons.ExecuteCommand(node, command, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in workload cluster")
	}

	// Scale CAPX to 2 replicas
	command = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system scale --replicas 2 deploy " + p.capxName + "-controller-manager"
	err = commons.ExecuteCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to scale CAPX in workload cluster")
	}

	// Allow egress in CAPX's Namespace
	command = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system apply -f " + allowAllEgressNetPolPath
	err = commons.ExecuteCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to apply CAPX's NetworkPolicy in workload cluster")
	}

	return nil
}

// installCAPXLocal installs CAPX in the local cluster
func (p *Provider) installCAPXLocal(node nodes.Node) error {
	var command string
	var err error

	command = "clusterctl init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +

		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	err = commons.ExecuteCommand(node, command, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in local cluster")
	}

	return nil
}

func enableSelfHealing(node nodes.Node, descriptorFile commons.DescriptorFile, namespace string) error {

	if !descriptorFile.ControlPlane.Managed {

		machineRole := "-control-plane-node"
		generateMHCManifest(node, descriptorFile.ClusterID, namespace, machineHealthCheckControlPlaneNodePath, machineRole)

		raw := bytes.Buffer{}
		cmd := node.Command("kubectl", "-n", namespace, "apply", "-f", machineHealthCheckControlPlaneNodePath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
		}
	}

	machineRole := "-worker-node"
	generateMHCManifest(node, descriptorFile.ClusterID, namespace, machineHealthCheckWorkerNodePath, machineRole)

	raw := bytes.Buffer{}
	cmd := node.Command("kubectl", "-n", namespace, "apply", "-f", machineHealthCheckWorkerNodePath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
	}

	return nil
}

func generateMHCManifest(node nodes.Node, clusterID string, namespace string, manifestPath string, machineRole string) error {

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

	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "echo \""+machineHealthCheck+"\" > "+manifestPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
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
		"base64": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
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
