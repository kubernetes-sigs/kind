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
	"encoding/base64"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

var defaultAWSSc = "gp2"

var storageClassAWSTemplate = StorageClassDef{
	APIVersion: "storage.k8s.io/v1",
	Kind:       "StorageClass",
	Metadata: struct {
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Name        string            `yaml:"name"`
	}{
		Annotations: map[string]string{
			"storageclass.kubernetes.io/is-default-class": "true",
		},
		Name: "keos",
	},
	Provisioner:       "ebs.csi.aws.com",
	Parameters:        make(map[string]interface{}),
	VolumeBindingMode: "WaitForFirstConsumer",
}

var standardAWSParameters = commons.SCParameters{
	Type: "gp2",
}

var premiumAWSParameters = commons.SCParameters{
	Type: "gp3",
}

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
	b.capxVersion = "v2.1.4"
	b.capxImageVersion = "2.1.4-0.4.0"
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

func (b *AWSBuilder) setCapxEnvVars(p commons.ProviderParams) {
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

func (b *AWSBuilder) getAzs(networks commons.Networks) ([]string, error) {
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
	if networks.Subnets != nil {
		privateAZs := []string{}
		for _, subnet := range networks.Subnets {
			privateSubnetID, _ := filterPrivateSubnet(svc, &subnet.SubnetId)
			if len(privateSubnetID) > 0 {
				sid := &ec2.DescribeSubnetsInput{
					SubnetIds: []*string{&subnet.SubnetId},
				}
				ds, err := svc.DescribeSubnets(sid)
				if err != nil {
					return nil, err
				}
				for _, describeSubnet := range ds.Subnets {
					if !slices.Contains(privateAZs, *describeSubnet.AvailabilityZone) {
						privateAZs = append(privateAZs, *describeSubnet.AvailabilityZone)
					}
				}
			}
		}
		return privateAZs, nil
	} else {
		result, err := svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
		if err != nil {
			return nil, err
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
}

func filterPrivateSubnet(svc *ec2.EC2, subnetID *string) (string, error) {
	keyname := "association.subnet-id"
	filters := make([]*ec2.Filter, 0)
	filter := ec2.Filter{
		Name: &keyname, Values: []*string{subnetID}}
	filters = append(filters, &filter)

	drti := &ec2.DescribeRouteTablesInput{Filters: filters}
	drto, err := svc.DescribeRouteTables(drti)
	if err != nil {
		return "", err
	}

	var isPublic bool
	for _, associatedRouteTable := range drto.RouteTables {
		for i := range associatedRouteTable.Routes {
			if *associatedRouteTable.Routes[i].DestinationCidrBlock == "0.0.0.0/0" &&
				associatedRouteTable.Routes[i].GatewayId != nil &&
				strings.Contains(*associatedRouteTable.Routes[i].GatewayId, "igw") {
				isPublic = true
			}
		}
	}
	if !isPublic {
		return *subnetID, nil
	} else {
		return "", nil
	}
}

func getEcrToken(p commons.ProviderParams) (string, error) {
	customProvider := credentials.NewStaticCredentialsProvider(
		p.Credentials["AccessKey"], p.Credentials["SecretKey"], "",
	)
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(customProvider),
		config.WithRegion(p.Region),
	)
	if err != nil {
		return "", err
	}

	svc := ecr.NewFromConfig(cfg)
	token, err := svc.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
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

func (b *AWSBuilder) configureStorageClass(n nodes.Node, k string, sc commons.StorageClass) error {
	var cmd exec.Cmd

	cmd = n.Command("kubectl", "--kubeconfig", k, "delete", "storageclass", defaultAWSSc)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to delete default StorageClass")
	}

	params := b.getParameters(sc)
	storageClass, err := insertParameters(storageClassAWSTemplate, params)
	if err != nil {
		return err
	}

	command := "sed -i 's/fsType/csi.storage.k8s.io\\/fstype/' " + storageClass
	_, err = commons.ExecuteCommand(n, command)
	if err != nil {
		return errors.Wrap(err, "failed to add csi.storage.k8s.io/fstype param to storageclass")
	}

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(storageClass)).Run(); err != nil {
		return errors.Wrap(err, "failed to create StorageClass")
	}
	return nil

}

func (b *AWSBuilder) getParameters(sc commons.StorageClass) commons.SCParameters {
	if sc.EncryptionKmsKey != "" {
		encrypted := true
		sc.Parameters.Encrypted = &encrypted
		sc.Parameters.KmsKeyId = sc.EncryptionKmsKey
	}
	switch class := sc.Class; class {
	case "standard":
		return mergeSCParameters(sc.Parameters, standardAWSParameters)
	case "premium":
		return mergeSCParameters(sc.Parameters, premiumAWSParameters)
	default:
		return standardAWSParameters
	}
}
