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
	_ "embed"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	CAPICoreProvider         = "cluster-api:v1.3.2"
	CAPIBootstrapProvider    = "kubeadm:v1.3.2"
	CAPIControlPlaneProvider = "kubeadm:v1.3.2"

	CNIName      = "calico"
	CNINamespace = "calico-system"
	CNIHelmChart = "/stratio/helm/tigera-operator"
	CNITemplate  = "/kind/calico-helm-values.yaml"
)

//go:embed files/calico-helm-values.yaml
var calicoHelmValues string

type PBuilder interface {
	setCapxProvider()
	setCapxName()
	setCapxTemplate(managed bool)
	setCapxEnvVars(p ProviderParams)
	setStorageClass()
	getProvider() Provider
}

type Provider struct {
	capxProvider string
	capxName     string
	capxTemplate string
	capxEnvVars  []string
	storageClass string
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
	return nil
}

func newInfra(b PBuilder) *Infra {
	return &Infra{
		builder: b,
	}
}

func (i *Infra) setBuilder(b PBuilder) {
	i.builder = b
}

func (i *Infra) buildProvider(p ProviderParams) Provider {
	i.builder.setCapxProvider()
	i.builder.setCapxName()
	i.builder.setCapxTemplate(p.managed)
	i.builder.setCapxEnvVars(p)
	i.builder.setStorageClass()
	return i.builder.getProvider()
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
		" --infrastructure " + p.capxProvider
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

	command = "clusterctl init --wait-providers" +
		" --core " + CAPICoreProvider +
		" --bootstrap " + CAPIBootstrapProvider +
		" --control-plane " + CAPIControlPlaneProvider +
		" --infrastructure " + p.capxProvider
	err = executeCommand(node, command, p.capxEnvVars)
	if err != nil {
		return errors.Wrap(err, "failed to install CAPX in local cluster")
	}

	return nil
}

func installCNI(node nodes.Node, kubeconfigPath string) error {
	var command string
	var err error

	command = "echo '" + calicoHelmValues + "' > " + CNITemplate
	err = executeCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to create CNI Helm chart values file")
	}

	command = "kubectl --kubeconfig " + kubeconfigPath + " create namespace " + CNINamespace
	err = executeCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to create CNI namespace")
	}

	command = "helm install --kubeconfig " + kubeconfigPath + " " + CNIName + " " + CNIHelmChart +
		" --namespace " + CNINamespace + " --values " + CNITemplate
	err = executeCommand(node, command)
	if err != nil {
		return errors.Wrap(err, "failed to deploy CNI Helm Chart")
	}

	return nil
}
