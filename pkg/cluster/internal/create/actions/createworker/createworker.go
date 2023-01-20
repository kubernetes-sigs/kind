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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
)

type action struct {
	vaultPassword string
}

// DescriptorFile represents the YAML structure in the cluster.yaml file
type DescriptorFile struct {
	ClusterID string `yaml:"cluster_id"`
	Keos      struct {
		Domain         string `yaml:"domain"`
		ExternalDomain string `yaml:"external_domain"`
		Flavour        string `yaml:"flavour"`
	} `yaml:"keos"`
	K8SVersion string  `yaml:"k8s_version"`
	Bastion    Bastion `yaml:"bastion"`
	Networks   struct {
		VPCID string `yaml:"vpc_id"`
	}
	ExternalRegistry map[string]interface{} `yaml:"external_registry"`
	//      ExternalRegistry     struct {
	//              AuthRequired    bool `yaml: auth_required`
	//              Type            string `yaml: type`
	//              URL             string `yaml: url`
	//      }
	Nodes struct {
		KubeNode struct {
			AmiID string `yaml:"ami_id"`
			Disks []struct {
				DeviceName string `yaml:"device_name"`
				Name       string `yaml:"name"`
				Path       string `yaml:"path,omitempty"`
				Size       int    `yaml:"size"`
				Type       string `yaml:"type"`
				Volumes    []struct {
					Name string `yaml:"name"`
					Path string `yaml:"path"`
					Size string `yaml:"size"`
				} `yaml:"volumes,omitempty"`
			} `yaml:"disks"`
			NodeType string `yaml:"node_type"`
			Quantity int    `yaml:"quantity"`
			VMSize   string `yaml:"vm_size"`
			Subnet   string `yaml:"subnet"`
			SSHKey   string `yaml:"ssh_key"`
			Spot     bool   `yaml:"spot"`
		} `yaml:"kube_node"`
	} `yaml:"nodes"`
	AWS AWS `yaml:"aws"`
	//B64Credentials string `yaml:"b64_credentials"`
	GithubToken string `yaml:"github_token"`
}

type SecretFile struct {
	Secrets struct {
		AWS AWS `yaml: "aws"`
	} `yaml: "secrets"`
}

type AWS struct {
	Credentials struct {
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		Region    string `yaml:"region"`
		Account   string `yaml:"account_id"`
		//AssumeRole string `yaml:"assume_role"`
	} `yaml:"credentials"`
}

// Bastion represents the bastion VM
type Bastion struct {
	AmiID             string   `yaml:"ami_id"`
	VMSize            string   `yaml:"vm_size"`
	AllowedCIDRBlocks []string `yaml:"allowedCIDRBlocks"`
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

// NewAction returns a new action for installing default CAPI
func NewAction(vaultPassword string) actions.Action {
	return &action{vaultPassword: vaultPassword}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {

	var aws AWS
	// var accessKey string
	// var account string
	// var region string
	// var secretKey string
	//var assumeRole string

	ctx.Status.Start("Installing CAPx in local 🎖️")
	defer ctx.Status.End(false)

	err := installCAPALocal(ctx, a.vaultPassword)
	if err != nil {
		return err
	}

	// mark success
	ctx.Status.End(true) // End Installing CAPx in local

	ctx.Status.Start("Generating worker cluster manifests 📝")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0] // kind expects at least one always

	// Read cluster.yaml file

	// descriptorRAW, err := os.ReadFile("./cluster.yaml")
	// if err != nil {
	// 	return err
	// }

	// var descriptorFile DescriptorFile
	// err = yaml.Unmarshal(descriptorRAW, &descriptorFile)
	// if err != nil {
	// 	return err
	// }

	// secretRaw, err := decryptFile("./secret.yaml", a.vaultPassword)
	// var secretFile SecretFile
	// if err != nil {
	// 	aws = descriptorFile.AWS
	// 	accessKey, account, region, secretKey = getCredentials(aws)
	// } else {
	// 	err = yaml.Unmarshal(stringToBytes(secretRaw), &secretFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	aws = secretFile.AWS
	// 	accessKey, account, region, secretKey = getCredentials(aws)
	// }

	descriptorRAW, err := os.ReadFile("./cluster.yaml")
	if err != nil {
		return err
	}

	var descriptorFile DescriptorFile
	err = yaml.Unmarshal(descriptorRAW, &descriptorFile)
	if err != nil {
		return err
	}

	aws, err = getCredentials(descriptorFile, a.vaultPassword)
	if err != nil {
		return err
	}

	// TODO STG: make k8s version configurable?

	capiClustersNamespace := "capi-clusters"

	// EKS specific: Generate the manifest
	descriptorData, err := generateEKSManifest(descriptorFile, capiClustersNamespace, aws)
	if err != nil {
		return errors.Wrap(err, "failed to generate EKS manifests")
	}

	// Create the cluster manifests file in the container
	descriptorPath := "/kind/manifests/cluster_" + descriptorFile.ClusterID + ".yaml"
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "echo \""+descriptorData+"\" > "+descriptorPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the cluster manifests")
	}

	ctx.Status.End(true) // End Generating worker cluster manifests

	ctx.Status.Start("Generating secrets file 📝🗝️")
	defer ctx.Status.End(false)

	filelines := []string{"secrets:\n", "  aws:\n", "    credentials:\n", "      access_key: " + aws.Credentials.AccessKey + "\n",
		"      account_id: " + aws.Credentials.Account + "\n", "      region: " + aws.Credentials.Region + "\n",
		"      secret_key: " + aws.Credentials.SecretKey + "\n"}

	basepath, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	directory := basepath
	err = createDirectory(directory)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	filename := directory + "/secrets.yaml"
	err = writeFile(filename, filelines)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = encryptFile(filename, a.vaultPassword)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	ctx.Status.Start("Creating the worker cluster 💥")
	defer ctx.Status.End(false)

	// Create namespace for CAPI clusters (it must exists)
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "create", "ns", capiClustersNamespace)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create cluster's Namespace")
	}
	// fmt.Println("RAW STRING: " + raw.String())

	// Apply cluster manifests
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "create", "-n", capiClustersNamespace, "-f", descriptorPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply manifests")
	}
	// fmt.Println("RAW STRING: " + raw.String())

	var machineHealthCheck = `
apiVersion: cluster.x-k8s.io/v1alpha3
kind: MachineHealthCheck
metadata:
  name: ` + descriptorFile.ClusterID + `-node-unhealthy
spec:
  clusterName: ` + descriptorFile.ClusterID + `
  nodeStartupTimeout: 120s
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
	machineHealthCheckPath := "/kind/machinehealthcheck.yaml"
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "echo \""+machineHealthCheck+"\" > "+machineHealthCheckPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the MachineHealthCheck manifest")
	}

	// Enable the cluster's self-healing
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "-n", capiClustersNamespace, "apply", "-f", machineHealthCheckPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply the MachineHealthCheck manifest")
	}

	// Wait for the worker cluster creation
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "-n", capiClustersNamespace, "wait", "--for=condition=ready", "--timeout", "25m", "cluster", descriptorFile.ClusterID)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create the worker Cluster")
	}

	// Wait for machines creation
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "-n", capiClustersNamespace, "wait", "--for=condition=ready", "--timeout", "20m", "--all", "md")
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create the Machines")
	}

	ctx.Status.End(true) // End Creating the worker cluster

	ctx.Status.Start("Installing CAPx in EKS 🎖️")
	defer ctx.Status.End(false)

	// Create the allow-all-egress network policy file in the container
	allowAllEgressNetPolPath := "/kind/allow-all-egress_netpol.yaml"
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "echo \""+allowAllEgressNetPol+"\" > "+allowAllEgressNetPolPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to write the allow-all-egress network policy")
	}

	// Get worker cluster's kubeconfig file (in EKS the token last 10m, which should be enough)
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "clusterctl -n "+capiClustersNamespace+" get kubeconfig "+descriptorFile.ClusterID+" > "+kubeconfigPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to get the kubeconfig file")
	}

	// AWS/EKS specific
	err = installCAPAWorker(aws, descriptorFile.GithubToken, node, kubeconfigPath, allowAllEgressNetPolPath)
	if err != nil {
		return err
	}

	//Scale CAPI to 2 replicas
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

	ctx.Status.End(true) // End Installing CAPx in worker cluster

	ctx.Status.Start("Transfering the management role 🗝️")
	defer ctx.Status.End(false)

	// Create namespace for CAPI clusters (it must exists) in worker cluster
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "create", "ns", capiClustersNamespace)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create manifests Namespace")
	}

	// EKS specific: Pivot management role to worker cluster
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "clusterctl move -n "+capiClustersNamespace+" --to-kubeconfig "+kubeconfigPath)
	cmd.SetEnv("AWS_REGION="+aws.Credentials.Region,
		"AWS_ACCESS_KEY_ID="+aws.Credentials.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+aws.Credentials.SecretKey,
		"AWS_B64ENCODED_CREDENTIALS="+generateB64Credentials(aws.Credentials.AccessKey, aws.Credentials.SecretKey, aws.Credentials.Region),
		"GITHUB_TOKEN="+descriptorFile.GithubToken)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to pivot management role to worker cluster")
	}

	ctx.Status.End(true) // End Transfering the management role

	return nil
}
