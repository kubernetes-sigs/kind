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
	b64 "encoding/base64"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

func getAWSEnv(region string, credentials map[string]string, githubToken string) []string {
	awsCredentials := "[default]\naws_access_key_id = " + credentials["AccessKey"] + "\naws_secret_access_key = " + credentials["SecretKey"] + "\nregion = " + region + "\n"
	envVars := []string{
		"AWS_REGION=" + region,
		"AWS_ACCESS_KEY_ID=" + credentials["AccessKey"],
		"AWS_SECRET_ACCESS_KEY=" + credentials["SecretKey"],
		"AWS_B64ENCODED_CREDENTIALS=" + b64.StdEncoding.EncodeToString([]byte(awsCredentials)),
		"GITHUB_TOKEN=" + githubToken,
		"CAPA_EKS_IAM=true"}
	return envVars
}

func installCAPXWorker(infraProvider string, node nodes.Node, envVars []string, capxName string, kubeconfigPath string, allowAllEgressNetPolPath string) error {

	// Install CAPX in worker cluster
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", "clusterctl --kubeconfig "+kubeconfigPath+" init --infrastructure "+infraProvider+" --wait-providers")
	cmd.SetEnv(envVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPX")
	}

	// Scale CAPX to 2 replicas
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "kubectl --kubeconfig "+kubeconfigPath+" -n "+capxName+"-system scale --replicas 2 deploy "+capxName+"-controller-manager")
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to scale the CAPA Deployment")
	}

	// Allow egress in CAPX's Namespace
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "kubectl --kubeconfig "+kubeconfigPath+" -n "+capxName+"-system apply -f "+allowAllEgressNetPolPath)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to apply CAPA's NetworkPolicy")
	}

	return nil
}

func createCloudFormationStack(node nodes.Node, envVars []string) error {
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
	return nil
}

// installCAPALocal installs CAPA in the local cluster
func installCAPXLocal(infraProvider string, node nodes.Node, envVars []string) error {
	var raw bytes.Buffer
	cmd := node.Command("sh", "-c", "clusterctl init --infrastructure "+infraProvider+" --wait-providers")
	cmd.SetEnv(envVars...)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPA")
	}
	return nil
}
