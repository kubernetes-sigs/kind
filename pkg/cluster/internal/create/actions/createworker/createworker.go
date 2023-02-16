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

// Package createworker implements the create worker action
package createworker

import (
	"bytes"
	"os"
	"strings"

	"github.com/gobeam/stringy"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/errors"
)

type action struct {
	vaultPassword  string
	descriptorName string
	moveManagement bool
	avoidCreation  bool
}

// SecretsFile represents the YAML structure in the secrets.yml file
type SecretsFile struct {
	Secrets struct {
		AWS struct {
			Credentials cluster.Credentials `yaml:"credentials"`
		} `yaml:"aws"`
		GCP struct {
			Credentials cluster.Credentials `yaml:"credentials"`
		} `yaml:"gcp"`
		GithubToken      string `yaml:"github_token"`
		ExternalRegistry struct {
			User string `yaml:"user"`
			Pass string `yaml:"pass"`
		} `yaml:"external_registry"`
	} `yaml:"secrets"`
}

const allowAllEgressNetPol = `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-all-egress
spec:
  egress:
  - {}
  podSelector: {}
  policyTypes:
  - Egress`

const kubeconfigPath = "/kind/worker-cluster.kubeconfig"
const workKubeconfigPath = ".kube/config"
const secretsFile = "secrets.yml"

// NewAction returns a new action for installing default CAPI
func NewAction(vaultPassword string, descriptorName string, moveManagement bool, avoidCreation bool) actions.Action {
	return &action{
		vaultPassword:  vaultPassword,
		descriptorName: descriptorName,
		moveManagement: moveManagement,
		avoidCreation:  avoidCreation,
	}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {

	// Get the target node
	node, err := getNode(ctx)
	if err != nil {
		return err
	}

	// Parse the cluster descriptor
	descriptorFile, err := cluster.GetClusterDescriptor(a.descriptorName)
	if err != nil {
		return errors.Wrap(err, "failed to parse cluster descriptor")
	}

	// Get the secrets
	credentials, externalRegistry, githubToken, err := getSecrets(*descriptorFile, a.vaultPassword)
	if err != nil {
		return err
	}

	providerParams := ProviderParams{
		region:      descriptorFile.Region,
		managed:     descriptorFile.ControlPlane.Managed,
		credentials: credentials,
		githubToken: githubToken,
	}

	providerBuilder := getBuilder(descriptorFile.InfraProvider)
	infra := newInfra(providerBuilder)
	provider := infra.buildProvider(providerParams)

	if descriptorFile.InfraProvider == "aws" {
		ctx.Status.Start("[CAPA] Ensuring IAM security 👮")
		defer ctx.Status.End(false)

		createCloudFormationStack(node, provider.capxEnvVars)

		ctx.Status.End(true) // End Ensuring CAPx requirements
	}

	ctx.Status.Start("Installing CAPx 🎖️")
	defer ctx.Status.End(false)

	err = provider.installCAPXLocal(node)
	if err != nil {
		return err
	}

	ctx.Status.End(true) // End Installing CAPx

	ctx.Status.Start("Generating workload cluster manifests 📝")
	defer ctx.Status.End(false)

	capiClustersNamespace := "cluster-" + descriptorFile.ClusterID

	templateParams := cluster.TemplateParams{
		Descriptor:       *descriptorFile,
		Credentials:      credentials,
		ExternalRegistry: externalRegistry,
	}

	// Generate the cluster manifest
	descriptorData, err := cluster.GetClusterManifest(provider.capxTemplate, templateParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate cluster manifests")
	}

	// Create the cluster manifests file in the container
	descriptorPath := "/kind/manifests/cluster_" + descriptorFile.ClusterID + ".yaml"
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "echo \""+descriptorData+"\" > "+descriptorPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the cluster manifests")
	}

	ctx.Status.End(true) // End Generating worker cluster manifests

	_, err = os.Stat(secretsFile)
	if err != nil {
		ctx.Status.Start("Generating secrets file 📝🗝️")
		defer ctx.Status.End(false)

		rewriteDescriptorFile(a.descriptorName)

		filelines := []string{
			"secrets:\n",
			"  github_token: " + githubToken + "\n",
			"  " + descriptorFile.InfraProvider + ":\n",
			"    credentials:\n",
		}

		for k, v := range credentials {
			if v != "" {
				v = strings.Replace(v, "\n", `\n`, -1)
				field := stringy.New(k)
				filelines = append(filelines, "      "+field.SnakeCase().ToLower()+": \""+v+"\"\n")
			}
		}

		filelines = append(filelines, "  external_registry:\n")
		filelines = append(filelines, "    user: "+externalRegistry["User"]+"\n")
		filelines = append(filelines, "    pass: "+externalRegistry["Pass"]+"\n")

		basepath, _ := currentdir()
		err = createDirectory(basepath)
		if err != nil {
			return err
		}
		filename := basepath + "/" + secretsFile
		err = writeFile(filename, filelines)
		if err != nil {
			return errors.Wrap(err, "failed to write the secrets file")
		}
		err = encryptFile(filename, a.vaultPassword)
		if err != nil {
			return errors.Wrap(err, "failed to cipher the secrets file")
		}

		defer ctx.Status.End(true) // End Generating secrets file
	}

	// Create namespace for CAPI clusters (it must exists)
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "create", "ns", capiClustersNamespace)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create cluster's Namespace")
	}

	var machineHealthCheck = `
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
metadata:
  name: ` + descriptorFile.ClusterID + `-node-unhealthy
spec:
  clusterName: ` + descriptorFile.ClusterID + `
  nodeStartupTimeout: 300s
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: ` + descriptorFile.ClusterID + `
  unhealthyConditions:
    - type: Ready
      status: Unknown
      timeout: 60s
    - type: Ready
      status: 'False'
      timeout: 60s`

	// Create the MachineHealthCheck manifest file in the container
	machineHealthCheckPath := "/kind/manifests/machinehealthcheck.yaml"
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "echo \""+machineHealthCheck+"\" > "+machineHealthCheckPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the MachineHealthCheck manifest")
	}

	// Create the allow-all-egress network policy file in the container
	allowAllEgressNetPolPath := "/kind/allow-all-egress_netpol.yaml"
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "echo \""+allowAllEgressNetPol+"\" > "+allowAllEgressNetPolPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the allow-all-egress network policy")
	}

	if !a.avoidCreation {

		ctx.Status.Start("Creating the workload cluster 💥")
		defer ctx.Status.End(false)

		// Apply cluster manifests
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "create", "-n", capiClustersNamespace, "-f", descriptorPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply manifests")
		}

		// Wait for the workload cluster control plane to be ready
		raw = bytes.Buffer{}
		cmd = node.Command("sh", "-c", "kubectl -n "+capiClustersNamespace+" wait --for=condition=ControlPlaneReady --timeout 20m cluster "+descriptorFile.ClusterID)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to wait for workload cluster Control Plane to be ready")
		}

		// Get the workload cluster kubeconfig
		raw = bytes.Buffer{}
		cmd = node.Command("sh", "-c", "clusterctl -n "+capiClustersNamespace+" get kubeconfig "+descriptorFile.ClusterID+" | tee "+kubeconfigPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to get workload cluster kubeconfig")
		}
		kubeconfig := raw.String()

		ctx.Status.End(true) // End Creating the workload cluster

		ctx.Status.Start("Saving the workload cluster kubeconfig 📝")
		defer ctx.Status.End(false)

		workKubeconfigBasePath := strings.Split(workKubeconfigPath, "/")[0]
		_, err = os.Stat(workKubeconfigBasePath)
		if err != nil {
			err := os.Mkdir(workKubeconfigBasePath, os.ModePerm)
			if err != nil {
				return err
			}
		}
		err = os.WriteFile(workKubeconfigPath, []byte(kubeconfig), 0600)
		if err != nil {
			return errors.Wrap(err, "failed to save the workload cluster kubeconfig")
		}

		ctx.Status.End(true) // End Saving the workload cluster kubeconfig

		// Install unmanaged cluster addons
		if !descriptorFile.ControlPlane.Managed {
			ctx.Status.Start("Installing CNI in workload cluster 🔌")
			defer ctx.Status.End(false)

			err = installCNI(node, kubeconfigPath)
			if err != nil {
				return errors.Wrap(err, "failed to install CNI in workload cluster")
			}
			ctx.Status.End(true) // End Installing CNI in workload cluster

			ctx.Status.Start("Installing StorageClass in workload cluster 💾")
			defer ctx.Status.End(false)

			err = infra.installCSI(node, kubeconfigPath)
			if err != nil {
				return errors.Wrap(err, "failed to install StorageClass in workload cluster")
			}
			ctx.Status.End(true) // End Installing StorageClass in workload cluster
		}

		ctx.Status.Start("Preparing nodes in workload cluster 📦")
		defer ctx.Status.End(false)

		// Wait for the worker cluster creation
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "-n", capiClustersNamespace, "wait", "--for=condition=ready", "--timeout", "10m", "cluster", descriptorFile.ClusterID)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to create the worker Cluster")
		}

		// Wait for machines creation
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "-n", capiClustersNamespace, "wait", "--for=condition=ready", "--timeout", "15m", "--all", "md")
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to create the Machines")
		}

		ctx.Status.End(true) // End Preparing nodes in workload cluster

		ctx.Status.Start("Enabling workload cluster's self-healing 🏥")
		defer ctx.Status.End(false)

		// Enable the cluster's self-healing
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "-n", capiClustersNamespace, "apply", "-f", machineHealthCheckPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
		}

		ctx.Status.End(true) // End Enabling workload cluster's self-healing

		ctx.Status.Start("Installing CAPx in workload cluster 🎖️")
		defer ctx.Status.End(false)

		err = provider.installCAPXWorker(node, kubeconfigPath, allowAllEgressNetPolPath)
		if err != nil {
			return err
		}

		// Scale CAPI to 2 replicas
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capi-system", "scale", "--replicas", "2", "deploy", "capi-controller-manager")
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to scale the CAPI Deployment")
		}

		// Allow egress in CAPI's Namespaces
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capi-system", "apply", "-f", allowAllEgressNetPolPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply CAPI's NetworkPolicy")
		}
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capi-kubeadm-bootstrap-system", "apply", "-f", allowAllEgressNetPolPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply CAPI's NetworkPolicy")
		}
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capi-kubeadm-control-plane-system", "apply", "-f", allowAllEgressNetPolPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply CAPI's NetworkPolicy")
		}

		// Allow egress in cert-manager Namespace
		raw = bytes.Buffer{}
		cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "cert-manager", "apply", "-f", allowAllEgressNetPolPath)
		if err := cmd.SetStdout(&raw).Run(); err != nil {
			return errors.Wrap(err, "failed to apply cert-manager's NetworkPolicy")
		}

		ctx.Status.End(true) // End Installing CAPx in workload cluster

		if descriptorFile.DeployAutoscaler {
			ctx.Status.Start("Adding Cluster-Autoescaler ⚙")
			defer ctx.Status.End(false)

			raw = bytes.Buffer{}
			cmd = integrateClusterAutoscaler(node, kubeconfigPath, descriptorFile.ClusterID, "clusterapi")
			if err := cmd.SetStdout(&raw).Run(); err != nil {
				return errors.Wrap(err, "failed to install chart cluster-autoscaler")
			}

			ctx.Status.End(true)
		}

		if !a.moveManagement {
			ctx.Status.Start("Moving the management role 🗝️")
			defer ctx.Status.End(false)

			// Create namespace for CAPI clusters (it must exists) in worker cluster
			raw = bytes.Buffer{}
			cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "create", "ns", capiClustersNamespace)
			if err := cmd.SetStdout(&raw).Run(); err != nil {
				return errors.Wrap(err, "failed to create manifests Namespace")
			}

			// Pivot management role to worker cluster
			raw = bytes.Buffer{}
			cmd = node.Command("sh", "-c", "clusterctl move -n "+capiClustersNamespace+" --to-kubeconfig "+kubeconfigPath)

			if err := cmd.SetStdout(&raw).Run(); err != nil {
				return errors.Wrap(err, "failed to pivot management role to worker cluster")
			}

			ctx.Status.End(true)
		}

		ctx.Status.Start("Generating the KEOS descriptor 📝")
		defer ctx.Status.End(false)

		err = createKEOSDescriptor(*descriptorFile, provider.storageClass)
		if err != nil {
			return err
		}
		ctx.Status.End(true) // End Generating KEOS descriptor

	}

	return nil
}
