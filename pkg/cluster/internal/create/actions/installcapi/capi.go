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

// Package installcapi implements the install CAPI action
package installcapi

import (
	"bytes"
	"os"

	"gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

type action struct{}

// SecretsFile represents the YAML structure in the secrets.yaml file
type SecretsFile struct {
	Secrets struct {
		AWS struct {
			Credentials struct {
				AccessKey string `yaml:"access_key"`
				SecretKey string `yaml:"secret_key"`
				Region    string `yaml:"region"`
				AccountID string `yaml:"account_id"`
			} `yaml:"credentials"`
			B64Credentials string `yaml:"b64_credentials"`
		} `yaml:"aws"`
		GithubToken string `yaml:"github_token"`
	} `yaml:"secrets"`
}

// NewAction returns a new action for installing default CAPI
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Ensuring IAM security 👮")
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

	// Right now, a CAPA pre-requisite is to have the region, aws_access_key_id and aws_secret_access_key
	// as environment variables. So we read the secrets.yaml file and ask for the decryption passphrase.
	// TODO STG: ask for the decryption passphrase (in new module "getcredentials"?)

	secretRAW, err := os.ReadFile("./secrets.yaml.clear")
	if err != nil {
		return err
	}

	var secretsFile SecretsFile
	err = yaml.Unmarshal(secretRAW, &secretsFile)
	if err != nil {
		return err
	}

	eksConfigData := `
apiVersion: bootstrap.aws.infrastructure.cluster.x-k8s.io/v1alpha1
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
	// - arn:aws:iam::` + secretsFile.Secrets.AWS.Credentials.AccountID + `:policy/csi.cluster-api-provider-aws.sigs.k8s.io`

	// fmt.Println("RAW STRING eksConfigData: " + eksConfigData)

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
	cmd = node.Command("clusterawsadm", "bootstrap", "iam", "create-cloudformation-stack", "--config", eksConfigPath)
	cmd.SetEnv("AWS_REGION="+secretsFile.Secrets.AWS.Credentials.Region,
		"AWS_ACCESS_KEY_ID="+secretsFile.Secrets.AWS.Credentials.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+secretsFile.Secrets.AWS.Credentials.SecretKey,
		"GITHUB_TOKEN="+secretsFile.Secrets.GithubToken)
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to run clusterawsadm")
	}
	// fmt.Println("RAW STRING: " + raw.String())
	// manifest := raw.String()
	ctx.Status.End(true) // End Ensuring CAPx requirements

	ctx.Status.Start("Installing CAPx in local 🎖️")
	defer ctx.Status.End(false)

	// Get credentials in B64
	// raw = bytes.Buffer{}
	// cmd = node.Command("clusterawsadm", "bootstrap", "credentials", "encode-as-profile")
	// cmd.SetEnv("AWS_REGION="+secretsFile.Secrets.AWS.Credentials.Region,
	// 	"AWS_ACCESS_KEY_ID="+secretsFile.Secrets.AWS.Credentials.AccessKey,
	// 	"AWS_SECRET_ACCESS_KEY="+secretsFile.Secrets.AWS.Credentials.SecretKey,
	// 	"GITHUB_TOKEN="+secretsFile.Secrets.GithubToken)
	// if err := cmd.SetStdout(&raw).Run(); err != nil {
	// 	return errors.Wrap(err, "failed to get clusterawsadm credentials")
	// }
	// // fmt.Println("RAW STRING: " + raw.String())
	// b64Credentials := strings.TrimSuffix(raw.String(), "\n")

	// Install CAPA
	raw = bytes.Buffer{}
	cmd = node.Command("sh", "-c", "clusterctl init --infrastructure aws --wait-providers")
	cmd.SetEnv("AWS_REGION="+secretsFile.Secrets.AWS.Credentials.Region,
		"AWS_ACCESS_KEY_ID="+secretsFile.Secrets.AWS.Credentials.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+secretsFile.Secrets.AWS.Credentials.SecretKey,
		"AWS_B64ENCODED_CREDENTIALS="+secretsFile.Secrets.AWS.B64Credentials,
		"GITHUB_TOKEN="+secretsFile.Secrets.GithubToken,
		"CAPA_EKS_IAM=true")
	// "EXP_MACHINE_POOL=true")
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to install CAPA")
	}

	// mark success
	ctx.Status.End(true) // End Installing CAPx

	return nil
}
