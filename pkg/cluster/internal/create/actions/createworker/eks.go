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

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

func getAWSEnv(region string, credentials map[string]string, githubToken string) []string {
	envVars := []string{
		"AWS_REGION=" + region,
		"AWS_ACCESS_KEY_ID=" + credentials["access_key"],
		"AWS_SECRET_ACCESS_KEY=" + credentials["secret_key"],
		"AWS_B64ENCODED_CREDENTIALS=" + generateB64Credentials(credentials["access_key"], credentials["secret_key"], region),
		"GITHUB_TOKEN=" + githubToken,
		"CAPA_EKS_IAM=true"}
	return envVars
}

// installCAPAWorker generates and apply the EKS manifests
func installCAPAWorker(node nodes.Node, envVars []string, kubeconfigPath string, allowAllEgressNetPolPath string) error {

	// Install CAPA in worker cluster
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "clusterctl --kubeconfig "+kubeconfigPath+" init --infrastructure aws --wait-providers")
	cmd.SetEnv(envVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPA")
	}

	//Scale CAPA to 2 replicas
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capa-system", "scale", "--replicas", "2", "deploy", "capa-controller-manager")
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to scale the CAPA Deployment")
	}

	// Allow egress in CAPA's Namespace
	raw = bytes.Buffer{}
	cmd = node.Command("kubectl", "--kubeconfig", kubeconfigPath, "-n", "capa-system", "apply", "-f", allowAllEgressNetPolPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply CAPA's NetworkPolicy")
	}

	return nil
}

// installCAPALocal installs CAPA in the local cluster
func installCAPALocal(node nodes.Node, ctx *actions.ActionContext, envVars []string) error {

	ctx.Status.Start("[CAPA] Ensuring IAM security 👮")
	defer ctx.Status.End(false)

	eksConfigData := `
apiVersion: bootstrap.aws.infrastructure.cluster.x-k8s.io/v1beta1
kind: AWSIAMConfiguration
spec:
  bootstrapUser:
    enable: true
  eks:
    enable: true
    iamRoleCreation: false
    defaultControlPlaneRole:
        disable: false
  controlPlane:
    enableCSIPolicy: true
  nodes:
    extraPolicyAttachments:
    - arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy`

	// Create the eks.config file in the container
	var raw bytes.Buffer
	eksConfigPath := "/kind/eks.config"
	cmd := node.Command("sh", "-c", "echo \""+eksConfigData+"\" > "+eksConfigPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to create eks.config")
	}

	// Run clusterawsadm with the eks.config file previously created
	// (this will create or update the CloudFormation stack in AWS)
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "clusterawsadm bootstrap iam create-cloudformation-stack --config "+eksConfigPath)
	cmd.SetEnv(envVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to run clusterawsadm")
	}
	ctx.Status.End(true) // End Ensuring CAPx requirements

	// Install CAPA
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "clusterctl init --infrastructure aws --wait-providers")
	cmd.SetEnv(envVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPA")
	}

	return nil
}
