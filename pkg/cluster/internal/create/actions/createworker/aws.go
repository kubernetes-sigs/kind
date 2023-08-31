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
	"context"
	_ "embed"
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed files/aws/internal-ingress-nginx.yaml
var awsInternalIngress []byte

type AWSBuilder struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxManaged      bool
	capxName         string
	capxTemplate     string
	capxEnvVars      []string
	scParameters     commons.SCParameters
	scProvisioner    string
	csiNamespace     string
}

func newAWSBuilder() *AWSBuilder {
	return &AWSBuilder{}
}

func (b *AWSBuilder) setCapx(managed bool) {
	b.capxProvider = "aws"
	b.capxVersion = "v2.2.1"
	b.capxImageVersion = "v2.2.1"
	b.capxName = "capa"
	b.capxManaged = managed
	b.csiNamespace = "kube-system"
	if managed {
		b.capxTemplate = "aws.eks.tmpl"
	} else {
		b.capxTemplate = "aws.tmpl"
	}
}

func (b *AWSBuilder) setCapxEnvVars(p ProviderParams) {
	awsCredentials := "[default]\naws_access_key_id = " + p.Credentials["AccessKey"] + "\naws_secret_access_key = " + p.Credentials["SecretKey"] + "\nregion = " + p.Region + "\n"
	b.capxEnvVars = []string{
		"AWS_REGION=" + p.Region,
		"AWS_ACCESS_KEY_ID=" + p.Credentials["AccessKey"],
		"AWS_SECRET_ACCESS_KEY=" + p.Credentials["SecretKey"],
		"AWS_B64ENCODED_CREDENTIALS=" + base64.StdEncoding.EncodeToString([]byte(awsCredentials)),
		"CAPA_EKS_IAM=true",
	}
	if p.GithubToken != "" {
		b.capxEnvVars = append(b.capxEnvVars, "GITHUB_TOKEN="+p.GithubToken)
	}
}

func (b *AWSBuilder) setSC(p ProviderParams) {
	if (p.StorageClass.Parameters != commons.SCParameters{}) {
		b.scParameters = p.StorageClass.Parameters
	}

	b.scProvisioner = "ebs.csi.aws.com"

	if b.scParameters.Type == "" {
		if p.StorageClass.Class == "premium" {
			b.scParameters.Type = "io2"
			b.scParameters.IopsPerGB = "64000"
		} else {
			b.scParameters.Type = "gp3"
		}
	}

	if p.StorageClass.EncryptionKey != "" {
		b.scParameters.Encrypted = "true"
		b.scParameters.KmsKeyId = p.StorageClass.EncryptionKey
	}
}

func (b *AWSBuilder) getProvider() Provider {
	return Provider{
		capxProvider:     b.capxProvider,
		capxVersion:      b.capxVersion,
		capxImageVersion: b.capxImageVersion,
		capxManaged:      b.capxManaged,
		capxName:         b.capxName,
		capxTemplate:     b.capxTemplate,
		capxEnvVars:      b.capxEnvVars,
		scParameters:     b.scParameters,
		scProvisioner:    b.scProvisioner,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AWSBuilder) installCSI(n nodes.Node, k string) error {
	var c string
	var err error

	c = "helm install aws-ebs-csi-driver /stratio/helm/aws-ebs-csi-driver" +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy AWS EBS CSI driver Helm Chart")
	}

	return nil
}

func createCloudFormationStack(n nodes.Node, envVars []string) error {
	var c string
	var err error

	eksConfigData := `
apiVersion: bootstrap.aws.infrastructure.cluster.x-k8s.io/v1beta1
kind: AWSIAMConfiguration
spec:
  bootstrapUser:
    enable: false
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
	eksConfigPath := "/kind/eks.config"
	c = "echo \"" + eksConfigData + "\" > " + eksConfigPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create eks.config")
	}

	// Run clusterawsadm with the eks.config file previously created (this will create or update the CloudFormation stack in AWS)
	c = "clusterawsadm bootstrap iam create-cloudformation-stack --config " + eksConfigPath
	_, err = commons.ExecuteCommand(n, c, envVars)
	if err != nil {
		return errors.Wrap(err, "failed to run clusterawsadm")
	}
	return nil
}

func (b *AWSBuilder) getAzs(p ProviderParams, networks commons.Networks) ([]string, error) {
	var err error
	var azs []string
	var ctx = context.TODO()

	cfg, err := commons.AWSGetConfig(ctx, p.Credentials, p.Region)
	if err != nil {
		return nil, err
	}
	svc := ec2.NewFromConfig(cfg)
	if len(networks.Subnets) > 0 {
		azs, err = commons.AWSGetPrivateAZs(ctx, svc, networks.Subnets)
		if err != nil {
			return nil, err
		}
	} else {
		azs, err = commons.AWSGetAZs(ctx, svc)
		if err != nil {
			return nil, err
		}
	}
	return azs, nil
}

func (b *AWSBuilder) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	var err error
	var ctx = context.TODO()

	cfg, err := commons.AWSGetConfig(ctx, p.Credentials, p.Region)
	if err != nil {
		return false, err
	}
	svc := ec2.NewFromConfig(cfg)
	if len(networks.Subnets) > 0 {
		for _, s := range networks.Subnets {
			isPrivate, err := commons.AWSIsPrivateSubnet(ctx, svc, &s.SubnetId)
			if err != nil {
				return false, err
			}
			if !isPrivate {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

func getEcrToken(p ProviderParams) (string, error) {
	var err error
	var ctx = context.TODO()

	cfg, err := commons.AWSGetConfig(ctx, p.Credentials, p.Region)
	if err != nil {
		return "", err
	}
	svc := ecr.NewFromConfig(cfg)
	token, err := svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}
	authData := token.AuthorizationData[0].AuthorizationToken
	data, err := base64.StdEncoding.DecodeString(*authData)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(string(data), ":", 2)
	return parts[1], nil
}

func (b *AWSBuilder) configureStorageClass(n nodes.Node, k string) error {
	var c string
	var err error
	var cmd exec.Cmd

	if b.capxManaged {
		// Remove annotation from default storage class
		c = "kubectl --kubeconfig " + k + ` get sc -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'`
		output, err := commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to get default storage class")
		}
		if strings.TrimSpace(output) != "" && strings.TrimSpace(output) != "No resources found" {
			c = "kubectl --kubeconfig " + k + " annotate sc " + strings.TrimSpace(output) + " " + defaultScAnnotation + "-"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to remove annotation from default storage class")
			}
		}
	}

	scTemplate.Parameters = b.scParameters
	scTemplate.Provisioner = b.scProvisioner

	scBytes, err := yaml.Marshal(scTemplate)
	if err != nil {
		return err
	}
	storageClass := strings.Replace(string(scBytes), "fsType", "csi.storage.k8s.io/fstype", -1)

	if b.scParameters.Labels != "" {
		var tags string
		re := regexp.MustCompile(`\s*labels: (.*,?)`)
		labels := re.FindStringSubmatch(storageClass)[1]
		for i, label := range strings.Split(labels, ",") {
			tags += "\n    tagSpecification_" + strconv.Itoa(i+1) + ": \"" + strings.TrimSpace(label) + "\""
		}
		storageClass = re.ReplaceAllString(storageClass, tags)
	}

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(storageClass)).Run(); err != nil {
		return errors.Wrap(err, "failed to create default storage class")
	}

	return nil
}

func (b *AWSBuilder) getOverrideVars(p ProviderParams, networks commons.Networks) (map[string][]byte, error) {
	var overrideVars map[string][]byte

	// Add override vars internal nginx
	requiredInternalNginx, err := b.internalNginx(p, networks)
	if err != nil {
		return nil, err
	}
	if requiredInternalNginx {
		overrideVars = addOverrideVar("ingress-nginx.yaml", awsInternalIngress, overrideVars)
	}

	// Add override vars for storage class
	if commons.Contains([]string{"io1", "io2"}, b.scParameters.Type) {
		overrideVars = addOverrideVar("storage-class.yaml", []byte("storage_class_pvc_size: 4Gi"), overrideVars)
	}
	if commons.Contains([]string{"st1", "sc1"}, b.scParameters.Type) {
		overrideVars = addOverrideVar("storage-class.yaml", []byte("storage_class_pvc_size: 125Gi"), overrideVars)
	}

	return overrideVars, nil
}
