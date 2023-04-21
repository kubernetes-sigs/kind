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
	b64 "encoding/base64"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

type AWSBuilder struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxName         string
	capxTemplate     string
	capxEnvVars      []string
	stClassName      string
	csiNamespace     string
}

func newAWSBuilder() *AWSBuilder {
	return &AWSBuilder{}
}

func (b *AWSBuilder) setCapx(managed bool) {
	b.capxProvider = "aws"
	b.capxVersion = "v2.0.2"
	b.capxImageVersion = "2.0.2-0.1.0"
	b.capxName = "capa"
	b.stClassName = "gp2"
	if managed {
		b.capxTemplate = "aws.eks.tmpl"
		b.csiNamespace = ""
	} else {
		b.capxTemplate = "aws.tmpl"
		b.csiNamespace = ""
	}
}

func (b *AWSBuilder) setCapxEnvVars(p ProviderParams) {
	awsCredentials := "[default]\naws_access_key_id = " + p.credentials["AccessKey"] + "\naws_secret_access_key = " + p.credentials["SecretKey"] + "\nregion = " + p.region + "\n"
	b.capxEnvVars = []string{
		"AWS_REGION=" + p.region,
		"AWS_ACCESS_KEY_ID=" + p.credentials["AccessKey"],
		"AWS_SECRET_ACCESS_KEY=" + p.credentials["SecretKey"],
		"AWS_B64ENCODED_CREDENTIALS=" + b64.StdEncoding.EncodeToString([]byte(awsCredentials)),
		"GITHUB_TOKEN=" + p.githubToken,
		"CAPA_EKS_IAM=true",
	}
}

func (b *AWSBuilder) getProvider() Provider {
	return Provider{
		capxProvider:     b.capxProvider,
		capxVersion:      b.capxVersion,
		capxImageVersion: b.capxImageVersion,
		capxName:         b.capxName,
		capxTemplate:     b.capxTemplate,
		capxEnvVars:      b.capxEnvVars,
		stClassName:      b.stClassName,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AWSBuilder) installCSI(n nodes.Node, k string) error {
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

func (b *AWSBuilder) getAzs() ([]string, error) {
	if len(b.capxEnvVars) == 0 {
		return nil, errors.New("Insufficient credentials.")
	}
	for _, cred := range b.capxEnvVars {
		c := strings.Split(cred, "=")
		envVar := c[0]
		envValue := c[1]
		os.Setenv(envVar, envValue)
	}

	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return nil, err
	}
	svc := ec2.New(sess)
	result, err := svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}
	if len(result.AvailabilityZones) < 3 {
		return nil, errors.New("Insufficient Availability Zones in this region. Must have at least 3")
	}
	azs := make([]string, 3)
	for i, az := range result.AvailabilityZones {
		if i == 3 {
			break
		}
		azs[i] = *az.ZoneName
	}
	return azs, nil
}
