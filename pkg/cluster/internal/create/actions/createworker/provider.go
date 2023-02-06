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

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

type PBuilder interface {
	setCapxProvider()
	setCapxName()
	setCapxTemplate(managed bool)
	setCapxEnvVars(p ProviderParams)
	getProvider() Provider
}

type Provider struct {
	capxProvider string
	capxName     string
	capxTemplate string
	capxEnvVars  []string
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
	return i.builder.getProvider()
}

// installCAPXWorker installs CAPX in the worker cluster
func installCAPXWorker(provider Provider, node nodes.Node, kubeconfigPath string, allowAllEgressNetPolPath string) error {

	// Install CAPX in worker cluster
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "clusterctl --kubeconfig "+kubeconfigPath+" init --infrastructure "+provider.capxProvider+" --wait-providers")
	cmd.SetEnv(provider.capxEnvVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPX")
	}

	// Scale CAPX to 2 replicas
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "kubectl --kubeconfig "+kubeconfigPath+" -n "+provider.capxName+"-system scale --replicas 2 deploy "+provider.capxName+"-controller-manager")
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to scale the CAPA Deployment")
	}

	// Allow egress in CAPX's Namespace
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "kubectl --kubeconfig "+kubeconfigPath+" -n "+provider.capxName+"-system apply -f "+allowAllEgressNetPolPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply CAPA's NetworkPolicy")
	}

	return nil
}

// installCAPXLocal installs CAPX in the local cluster
func installCAPXLocal(provider Provider, node nodes.Node) error {
	var raw bytes.Buffer
	cmd := node.Command("sh", "-c", "clusterctl init --infrastructure "+provider.capxProvider+" --wait-providers")
	cmd.SetEnv(provider.capxEnvVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPX")
	}
	return nil
}
