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
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/exp/slices"
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
	b.capxVersion = "v2.2.0"
	b.capxImageVersion = "v2.2.0"
	b.capxName = "capa"

	b.csiNamespace = "kube-system"

	if managed {
		b.capxManaged = true
		b.capxTemplate = "aws.eks.tmpl"
	} else {
		b.capxManaged = false
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

func (b *AWSBuilder) internalNginx(networks commons.Networks, credentialsMap map[string]string, clusterName string) (bool, error) {
	if len(b.capxEnvVars) == 0 {
		return false, errors.New("Insufficient credentials.")
	}
	for _, cred := range b.capxEnvVars {
		c := strings.Split(cred, "=")
		envVar := c[0]
		envValue := c[1]
		os.Setenv(envVar, envValue)
	}

	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return false, err
	}
	svc := ec2.New(sess)
	if networks.Subnets != nil {
		for _, subnet := range networks.Subnets {
			publicSubnetID, _ := filterPublicSubnet(svc, &subnet.SubnetId)
			if len(publicSubnetID) > 0 {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
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
			route := associatedRouteTable.Routes[i]

			if route.DestinationCidrBlock != nil &&
				route.GatewayId != nil &&
				*route.DestinationCidrBlock == "0.0.0.0/0" &&
				strings.Contains(*route.GatewayId, "igw") {
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

func filterPublicSubnet(svc *ec2.EC2, subnetID *string) (string, error) {
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
			route := associatedRouteTable.Routes[i]

			if route.DestinationCidrBlock != nil &&
				route.GatewayId != nil &&
				*route.DestinationCidrBlock == "0.0.0.0/0" &&
				strings.Contains(*route.GatewayId, "igw") {
				isPublic = true
			}
		}
	}
	if isPublic {
		return *subnetID, nil
	} else {
		return "", nil
	}
}

func getEcrToken(p ProviderParams) (string, error) {
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

func (b *AWSBuilder) getOverrideVars(keosCluster commons.KeosCluster, credentialsMap map[string]string) (map[string][]byte, error) {
	overrideVars := map[string][]byte{}
	InternalNginxOVPath, InternalNginxOVValue, err := b.getInternalNginxOverrideVars(keosCluster.Spec.Networks, credentialsMap, keosCluster.Metadata.Name)
	if err != nil {
		return nil, err
	}

	overrideVars = addOverrideVar(InternalNginxOVPath, InternalNginxOVValue, overrideVars)

	// Add override vars for storage class
	if commons.Contains([]string{"io1", "io2"}, b.scParameters.Type) {
		overrideVars = addOverrideVar("storage-class.yaml", []byte("storage_class_pvc_size: 4Gi"), overrideVars)
	}
	if commons.Contains([]string{"st1", "sc1"}, b.scParameters.Type) {
		overrideVars = addOverrideVar("storage-class.yaml", []byte("storage_class_pvc_size: 125Gi"), overrideVars)
	}

	return overrideVars, nil
}

func (b *AWSBuilder) getInternalNginxOverrideVars(networks commons.Networks, credentialsMap map[string]string, clusterName string) (string, []byte, error) {
	requiredInternalNginx, err := b.internalNginx(networks, credentialsMap, clusterName)
	if err != nil {
		return "", nil, err
	}

	if requiredInternalNginx {
		return "ingress-nginx.yaml", awsInternalIngress, nil
	}

	return "", []byte(""), nil
}
