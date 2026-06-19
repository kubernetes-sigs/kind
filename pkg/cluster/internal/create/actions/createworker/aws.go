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

//go:embed files/aws/public-ingress-nginx.yaml
var awsPublicIngress []byte

type AWSBuilder struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxManaged      bool
	capxName         string
	capxEnvVars      []string
	scParameters     commons.SCParameters
	scProvisioner    string
	csiNamespace     string
}

type lbControllerHelmParams struct {
	ClusterName string
	Private     bool
	KeosRegUrl  string
	AccountID   string
	RoleName    string
}

func newAWSBuilder() *AWSBuilder {
	return &AWSBuilder{}
}

func (b *AWSBuilder) setCapx(managed bool, capx commons.CAPX) {
	b.capxProvider = "aws"
	b.capxVersion = capx.CAPA_Version
	b.capxImageVersion = capx.CAPA_Image_version
	b.capxName = "capa"
	b.capxManaged = managed
	b.csiNamespace = "kube-system"
}

func (b *AWSBuilder) setCapxEnvVars(p ProviderParams) {
	awsCredentials := "[default]\naws_access_key_id = " + p.Credentials["AccessKey"] + "\naws_secret_access_key = " + p.Credentials["SecretKey"] + "\nregion = " + p.Region + "\n"
	// Add ROLE_ARN to awsCredentials if present and not "false"
	if p.Credentials["RoleARN"] != "" {
		awsCredentials += "role_arn = " + p.Credentials["RoleARN"] + "\n"
	}
	b.capxEnvVars = []string{
		"AWS_REGION=" + p.Region,
		"AWS_ACCESS_KEY_ID=" + p.Credentials["AccessKey"],
		"AWS_SECRET_ACCESS_KEY=" + p.Credentials["SecretKey"],
		// AWS_B64ENCODED_CREDENTIALS will be the content of secret "keoscluster-settings"
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

	// Enable encryption by default with aws/ebs (AWS managed key if no kmsKeyId provided)
	b.scParameters.Encrypted = "true"
	if p.StorageClass.EncryptionKey != "" {
		b.scParameters.KmsKeyId = p.StorageClass.EncryptionKey
	}
}

var awsCharts = ChartsDictionary{
	Charts: map[string]map[string]map[string]commons.ChartEntry{
		"32": {
			"managed": {
				"aws-load-balancer-controller": {Repository: "https://aws.github.io/eks-charts", Version: "1.14.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"cluster-autoscaler":           {Repository: "https://kubernetes.github.io/autoscaler", Version: "9.52.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"tigera-operator":              {Repository: "https://docs.projectcalico.org/charts", Version: "v3.30.2", Namespace: "tigera-operator", Pull: true, Reconcile: true},
			},
			"unmanaged": {},
		},
		"33": {
			"managed": {
				"aws-load-balancer-controller": {Repository: "https://aws.github.io/eks-charts", Version: "1.14.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"cluster-autoscaler":           {Repository: "https://kubernetes.github.io/autoscaler", Version: "9.52.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"tigera-operator":              {Repository: "https://docs.projectcalico.org/charts", Version: "v3.30.2", Namespace: "tigera-operator", Pull: true, Reconcile: true},
			},
			"unmanaged": {},
		},
		"34": {
			"managed": {
				"aws-load-balancer-controller": {Repository: "https://aws.github.io/eks-charts", Version: "1.14.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"cluster-autoscaler":           {Repository: "https://kubernetes.github.io/autoscaler", Version: "9.52.1", Namespace: "kube-system", Pull: false, Reconcile: false},
				"tigera-operator":              {Repository: "https://docs.projectcalico.org/charts", Version: "v3.30.2", Namespace: "tigera-operator", Pull: true, Reconcile: true},
			},
			"unmanaged": {},
		},
		"35": {
			"managed": {
				"aws-load-balancer-controller": {Repository: "https://aws.github.io/eks-charts", Version: "3.4.0", Namespace: "kube-system", Pull: false, Reconcile: false},
				"cluster-autoscaler":           {Repository: "https://kubernetes.github.io/autoscaler", Version: "9.57.0", Namespace: "kube-system", Pull: false, Reconcile: false},
				"tigera-operator":              {Repository: "https://docs.projectcalico.org/charts", Version: "v3.31.5", Namespace: "tigera-operator", Pull: true, Reconcile: true},
			},
			"unmanaged": {},
		},
	},
}

func (b *AWSBuilder) pullProviderCharts(n nodes.Node, clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterCredentials commons.ClusterCredentials, clusterType string) error {
	if clusterConfigSpec.EKSLBController && clusterType == "managed" {
		for name, chart := range awsCharts.Charts[majorVersion][clusterType] {
			if name == "aws-load-balancer-controller" {
				chart.Pull = true
				awsCharts.Charts[majorVersion][clusterType][name] = chart
			}
		}
	}
	return pullGenericCharts(n, clusterConfigSpec, keosSpec, clusterCredentials, awsCharts, clusterType)

}

func (b *AWSBuilder) getProviderCharts(clusterConfigSpec *commons.ClusterConfigSpec, keosSpec commons.KeosSpec, clusterType string) map[string]commons.ChartEntry {
	return getGenericCharts(clusterConfigSpec, keosSpec, awsCharts, clusterType)
}

func (b *AWSBuilder) getOverriddenCharts(charts *[]commons.Chart, clusterConfigSpec *commons.ClusterConfigSpec, clusterType string) []commons.Chart {
	providerCharts := ConvertToChart(awsCharts.Charts[majorVersion][clusterType])
	for _, ovChart := range clusterConfigSpec.Charts {
		for _, chart := range *providerCharts {
			if chart.Name == ovChart.Name {
				chart.Version = ovChart.Version
			}
		}
	}
	*charts = append(*charts, *providerCharts...)
	return *charts
}

func (b *AWSBuilder) getProvider() Provider {
	return Provider{
		capxProvider:     b.capxProvider,
		capxVersion:      b.capxVersion,
		capxImageVersion: b.capxImageVersion,
		capxManaged:      b.capxManaged,
		capxName:         b.capxName,
		capxEnvVars:      b.capxEnvVars,
		scParameters:     b.scParameters,
		scProvisioner:    b.scProvisioner,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AWSBuilder) installCloudProvider(n nodes.Node, k string, privateParams PrivateParams) error {
	return nil
}

func (b *AWSBuilder) installCSI(n nodes.Node, k string, privateParams PrivateParams, providerParams ProviderParams, chartsList map[string]commons.ChartEntry) error {
	return nil
}

func installLBController(n nodes.Node, k string, privateParams PrivateParams, p ProviderParams, chartsList map[string]commons.ChartEntry) error {
	lbControllerName := "aws-load-balancer-controller"
	lbControllerValuesFile := "/kind/" + lbControllerName + "-helm-values.yaml"
	lbControllerEntry := chartsList[lbControllerName]
	clusterName := p.ClusterName
	roleName := clusterName + "-lb-controller-manager"
	accountID := p.Credentials["AccountID"]

	lbControllerManagerHelmParams := lbControllerHelmParams{
		ClusterName: privateParams.KeosCluster.Metadata.Name,
		Private:     privateParams.Private,
		KeosRegUrl:  commons.GetPrefixedRegistryURL("public.ecr.aws", privateParams.KeosRegUrl, privateParams.CentralECR),
		AccountID:   accountID,
		RoleName:    roleName,
	}

	lbControllerHelmReleaseParams := fluxHelmReleaseParams{
		HelmReleaseName: lbControllerName,
		ChartRepoRef:    "keos",
		ChartName:       lbControllerName,
		ChartNamespace:  lbControllerEntry.Namespace,
		ChartVersion:    lbControllerEntry.Version,
	}
	if !privateParams.HelmPrivate {
		lbControllerHelmReleaseParams.ChartRepoRef = lbControllerName
	}
	// Generate the aws lb controller helm values
	lbControllerHelmValues, getManifestErr := getManifest(privateParams.KeosCluster.Spec.InfraProvider, lbControllerName+"-helm-values.tmpl", majorVersion, lbControllerManagerHelmParams)
	if getManifestErr != nil {
		return errors.Wrap(getManifestErr, "failed to generate "+lbControllerName+"-csi helm values")
	}

	// Add clusterName to the Helm values
	lbControllerHelmValues += "\nclusterName: " + clusterName

	c := "echo '" + lbControllerHelmValues + "' > " + lbControllerValuesFile
	_, err := commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create "+lbControllerName+" Helm chart values file")
	}
	if err := configureHelmRelease(n, kubeconfigPath, "flux2_helmrelease.tmpl", lbControllerHelmReleaseParams, privateParams.KeosCluster.Spec.HelmRepository); err != nil {
		return err
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
    managedMachinePool:
        disable: false
        extraPolicyAttachments:
        - arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy
  controlPlane:
    enableCSIPolicy: true
  nodes:
    extraPolicyAttachments:
    - arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy
    extraStatements:
    - Action:
        - "ecr:BatchImportUpstreamImage"
        - "ecr:CreateRepository"
      Effect: Allow
      Resource:
        - "*"`

	// Create the eks.config file in the container
	eksConfigPath := "/kind/eks.config"
	c = "echo '" + eksConfigData + "' > " + eksConfigPath

	_, err = commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		return errors.Wrap(err, "failed to create eks.config")
	}

	// Run clusterawsadm with the eks.config file previously created (this will create or update the CloudFormation stack in AWS)
	c = "clusterawsadm bootstrap iam create-cloudformation-stack --config " + eksConfigPath

	_, err = commons.ExecuteCommand(n, c, 5, 3, envVars)
	if err != nil {
		return errors.Wrap(err, "failed to run clusterawsadm")
	}
	return nil
}

func (b *AWSBuilder) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	var err error
	var ctx = context.TODO()

	cfg, err := commons.AWSGetConfig(ctx, p.Credentials)
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

func (b *AWSBuilder) getRegistryCredentials(p ProviderParams, u string) (string, string, error) {
	var registryUser = "AWS"
	var registryPass string
	var ctx = context.Background()

	cfg, err := commons.AWSGetConfig(ctx, p.Credentials)
	if err != nil {
		return "", "", err
	}
	svc := ecr.NewFromConfig(cfg)
	token, err := svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", "", err
	}
	authData := token.AuthorizationData[0].AuthorizationToken
	data, err := base64.StdEncoding.DecodeString(*authData)
	if err != nil {
		return "", "", err
	}
	registryPass = strings.SplitN(string(data), ":", 2)[1]
	return registryUser, registryPass, nil
}

func (b *AWSBuilder) configureStorageClass(n nodes.Node, k string) error {
	var c string
	var err error
	var cmd exec.Cmd

	if b.capxManaged {
		// Remove annotation from default storage class
		c = "kubectl --kubeconfig " + k + ` get sc -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'`

		output, err := commons.ExecuteCommand(n, c, 5, 3)
		if err != nil {
			return errors.Wrap(err, "failed to get default storage class")
		}
		if strings.TrimSpace(output) != "" && strings.TrimSpace(output) != "No resources found" {
			c = "kubectl --kubeconfig " + k + " annotate sc " + strings.TrimSpace(output) + " " + defaultScAnnotation + "-"

			_, err = commons.ExecuteCommand(n, c, 5, 3)
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

func (b *AWSBuilder) getOverrideVars(p ProviderParams, networks commons.Networks, clusterConfigSpec commons.ClusterConfigSpec) (map[string][]byte, error) {
	var overrideVars = make(map[string][]byte)

	// Add override vars internal nginx
	requiredInternalNginx, err := b.internalNginx(p, networks)
	if err != nil {
		return nil, err
	}
	if requiredInternalNginx {
		overrideVars = addOverrideVar("ingress-nginx.yaml", awsInternalIngress, overrideVars)
	} else if !requiredInternalNginx && p.Managed && clusterConfigSpec.EKSLBController {
		overrideVars = addOverrideVar("ingress-nginx.yaml", awsPublicIngress, overrideVars)
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

func (b *AWSBuilder) postInstallPhase(n nodes.Node, k string) error {
	var coreDNSPDBName = "coredns"

	c := "kubectl --kubeconfig " + kubeconfigPath + " get pdb " + coreDNSPDBName + " -n kube-system"

	_, err := commons.ExecuteCommand(n, c, 5, 3)
	if err != nil {
		err = installCorednsPdb(n)
		if err != nil {
			return errors.Wrap(err, "failed to add core dns PDB")
		}
	}
	if b.capxManaged {
		err := patchDeploy(n, k, "kube-system", "coredns", "{\"spec\": {\"template\": {\"metadata\": {\"annotations\": {\""+postInstallAnnotation+"\": \"tmp\"}}}}}")
		if err != nil {
			return errors.Wrap(err, "failed to add podAnnotation to coredns")
		}

		err = patchDeploy(n, k, "kube-system", "ebs-csi-controller", "{\"spec\": {\"template\": {\"metadata\": {\"annotations\": {\""+postInstallAnnotation+"\": \"socket-dir\"}}}}}")
		if err != nil {
			return errors.Wrap(err, "failed to add podAnnotation to ebs-csi-controller")
		}
	}

	return nil
}
