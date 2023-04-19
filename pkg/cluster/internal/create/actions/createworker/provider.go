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
	_ "embed"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	CAPICoreProvider         = "cluster-api:v1.3.2"
	CAPIBootstrapProvider    = "kubeadm:v1.3.2"
	CAPIControlPlaneProvider = "kubeadm:v1.3.2"
	//CAPILocalRepository      = "/root/.cluster-api/local-repository"

	CalicoName      = "calico"
	CalicoNamespace = "calico-system"
	CalicoHelmChart = "/stratio/helm/tigera-operator"
	CalicoTemplate  = "/kind/calico-helm-values.yaml"
)

const machineHealthCheckWorkerNodePath = "/kind/manifests/machinehealthcheckworkernode.yaml"
const machineHealthCheckControlPlaneNodePath = "/kind/manifests/machinehealthcheckcontrolplane.yaml"

type PBuilder interface {
	setCapx(managed bool)
	setCapxEnvVars(p ProviderParams)
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

type ProviderParams struct {
	region      string
	managed     bool
	credentials map[string]string
	githubToken string
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
	i.builder.setCapx(p.managed)
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

func installCalico(n nodes.Node, k string, descriptorFile cluster.DescriptorFile) error {
	var c string
	var err error

	// Generate the calico helm values
	calicoHelmValues, err := getCalicoManifest(descriptorFile)
	if err != nil {
		return errors.Wrap(err, "failed to generate calico helm values")
	}

	c = "kubectl --kubeconfig " + k + " create namespace " + CalicoNamespace
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create Calico namespace")
	}

	c = "echo '" + calicoHelmValues + "' > " + CalicoTemplate
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create Calico Helm chart values file")
	}

	c = "helm install --kubeconfig " + k + " " + CalicoName + " " + CalicoHelmChart +
		" --namespace " + CalicoNamespace + " --values " + CalicoTemplate
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Calico Helm Chart")
	}

	return nil
}

// installCAPXWorker installs CAPX in the worker cluster
func (p *Provider) installCAPXWorker(node nodes.Node, kubeconfigPath string, allowAllEgressNetPolPath string) error {
	var command string
	var err error

	if p.capxProvider == "azure" {
		// Create capx namespace
		command = "kubectl --kubeconfig " + kubeconfigPath + " create namespace " + p.capxName + "-system"
		err = executeCommand(node, command)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		secret := strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET=")[1]
		command = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system create secret generic cluster-identity-secret --from-literal=clientSecret='" + string(secret) + "'"
		err = executeCommand(node, command)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	// Install CAPX in worker cluster
	command = "clusterctl --kubeconfig " + kubeconfigPath + " init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	err = executeCommand(node, command, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in workload cluster")
	}

	// Scale CAPX to 2 replicas
	command = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system scale --replicas 2 deploy " + p.capxName + "-controller-manager"
	err = executeCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to scale CAPX in workload cluster")
	}

	// Allow egress in CAPX's Namespace
	command = "kubectl --kubeconfig " + kubeconfigPath + " -n " + p.capxName + "-system apply -f " + allowAllEgressNetPolPath
	err = executeCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to apply CAPX's NetworkPolicy in workload cluster")
	}

	return nil
}

// installCAPXLocal installs CAPX in the local cluster
func (p *Provider) installCAPXLocal(node nodes.Node) error {
	var command string
	var err error

	if p.capxProvider == "azure" {
		// Create capx namespace
		command = "kubectl create namespace " + p.capxName + "-system"
		err = executeCommand(node, command)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx namespace")
		}

		// Create capx secret
		secret := strings.Split(p.capxEnvVars[0], "AZURE_CLIENT_SECRET=")[1]
		command = "kubectl -n " + p.capxName + "-system create secret generic cluster-identity-secret --from-literal=clientSecret='" + string(secret) + "'"
		err = executeCommand(node, command)
		if err != nil {
			return errors.Wrap(err, "failed to create CAPx secret")
		}
	}

	command = "clusterctl init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider + ":" + p.capxVersion
	err = executeCommand(node, command, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in local cluster")
	}

	return nil
}

func enableSelfHealing(node nodes.Node, descriptorFile cluster.DescriptorFile, namespace string) error {

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
