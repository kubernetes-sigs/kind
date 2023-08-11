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
	"context"
	_ "embed"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

type action struct {
	vaultPassword      string
	descriptorPath     string
	moveManagement     bool
	avoidCreation      bool
	keosCluster        commons.KeosCluster
	clusterCredentials commons.ClusterCredentials
}

type keosRegistry struct {
	url          string
	user         string
	pass         string
	registryType string
}

const (
	kubeconfigPath          = "/kind/worker-cluster.kubeconfig"
	workKubeconfigPath      = ".kube/config"
	CAPILocalRepository     = "/root/.cluster-api/local-repository"
	cloudProviderBackupPath = "/kind/backup/objects"
	localBackupPath         = "backup"
	manifestsPath           = "/kind/manifests"

	keosClusterVersion = "0.1.0-SNAPSHOT"
)

var PathsToBackupLocally = []string{
	cloudProviderBackupPath,
	"/kind/manifests",
}

//go:embed files/common/allow-all-egress_netpol.yaml
var allowCommonEgressNetPol string

//go:embed files/gcp/rbac-loadbalancing.yaml
var rbacInternalLoadBalancing string

// NewAction returns a new action for installing default CAPI
func NewAction(vaultPassword string, descriptorPath string, moveManagement bool, avoidCreation bool, keosCluster commons.KeosCluster, clusterCredentials commons.ClusterCredentials) actions.Action {
	return &action{
		vaultPassword:      vaultPassword,
		descriptorPath:     descriptorPath,
		moveManagement:     moveManagement,
		avoidCreation:      avoidCreation,
		keosCluster:        keosCluster,
		clusterCredentials: clusterCredentials,
	}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	var c string
	var err error
	var keosRegistry keosRegistry
	var jsonDockerRegistriesCredentials []byte

	// Get the target node
	n, err := ctx.GetNode()
	if err != nil {
		return err
	}

	providerParams := ProviderParams{
		ClusterName:  a.keosCluster.Metadata.Name,
		Region:       a.keosCluster.Spec.Region,
		Managed:      a.keosCluster.Spec.ControlPlane.Managed,
		Credentials:  a.clusterCredentials.ProviderCredentials,
		GithubToken:  a.clusterCredentials.GithubToken,
		StorageClass: a.keosCluster.Spec.StorageClass,
	}

	providerBuilder := getBuilder(a.keosCluster.Spec.InfraProvider)
	infra := newInfra(providerBuilder)
	provider := infra.buildProvider(providerParams)

	awsEKSEnabled := a.keosCluster.Spec.InfraProvider == "aws" && a.keosCluster.Spec.ControlPlane.Managed
	azureAKSEnabled := a.keosCluster.Spec.InfraProvider == "azure" && a.keosCluster.Spec.ControlPlane.Managed

	ctx.Status.Start("Installing CAPx 🎖️")
	defer ctx.Status.End(false)

	for _, registry := range a.keosCluster.Spec.DockerRegistries {
		if registry.KeosRegistry {
			keosRegistry.url = registry.URL
			keosRegistry.registryType = registry.Type
			continue
		}
	}

	if keosRegistry.registryType == "ecr" {
		ecrToken, err := getEcrToken(providerParams)
		if err != nil {
			return errors.Wrap(err, "failed to get ECR auth token")
		}
		keosRegistry.user = "AWS"
		keosRegistry.pass = ecrToken
	} else if keosRegistry.registryType == "acr" {
		acrService := strings.Split(keosRegistry.url, "/")[0]
		acrToken, err := getAcrToken(providerParams, acrService)
		if err != nil {
			return errors.Wrap(err, "failed to get ACR auth token")
		}
		keosRegistry.user = "00000000-0000-0000-0000-000000000000"
		keosRegistry.pass = acrToken
	} else {
		keosRegistry.user = a.clusterCredentials.KeosRegistryCredentials["User"]
		keosRegistry.pass = a.clusterCredentials.KeosRegistryCredentials["Pass"]
	}

	// Create docker-registry secret for keos cluster
	c = "kubectl -n kube-system create secret docker-registry regcred" +
		" --docker-server=" + keosRegistry.url +
		" --docker-username=" + keosRegistry.user +
		" --docker-password=" + keosRegistry.pass
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create docker-registry secret")
	}

	if provider.capxVersion != provider.capxImageVersion {

		infraComponents := CAPILocalRepository + "/infrastructure-" + provider.capxProvider + "/" + provider.capxVersion + "/infrastructure-components.yaml"

		// Create provider-system namespace
		c = "kubectl create namespace " + provider.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create "+provider.capxName+"-system namespace")
		}

		// Create docker-registry secret in provider-system namespace
		c = "kubectl create secret docker-registry regcred" +
			" --docker-server=" + keosRegistry.url +
			" --docker-username=" + keosRegistry.user +
			" --docker-password=" + keosRegistry.pass +
			" --namespace=" + provider.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create docker-registry secret")
		}

		// Add imagePullSecrets to infrastructure-components.yaml
		c = "sed -i '/containers:/i\\      imagePullSecrets:\\n      - name: regcred' " + infraComponents
		_, err = commons.ExecuteCommand(n, c)

		if err != nil {
			return errors.Wrap(err, "failed to add imagePullSecrets to infrastructure-components.yaml")
		}
	}

	err = provider.installCAPXLocal(n)
	if err != nil {
		return err
	}

	ctx.Status.End(true) // End Installing CAPx

	ctx.Status.Start("Generating workload cluster manifests 📝")
	defer ctx.Status.End(false)

	capiClustersNamespace := "cluster-" + a.keosCluster.Metadata.Name

	azs, err := infra.getAzs(providerParams, a.keosCluster.Spec.Networks)
	if err != nil {
		return errors.Wrap(err, "failed to get AZs")
	}

	templateParams := commons.TemplateParams{
		KeosCluster:      a.keosCluster,
		Credentials:      a.clusterCredentials.ProviderCredentials,
		DockerRegistries: a.clusterCredentials.DockerRegistriesCredentials,
		AZs:              azs,
		Flavor:           provider.capxTemplate,
	}

	// Generate the cluster manifest
	descriptorData, err := GetClusterManifest(templateParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate cluster manifests")
	}

	// Create the cluster manifests file in the container
	descriptorPath := manifestsPath + "/cluster_" + a.keosCluster.Metadata.Name + ".yaml"
	c = "echo \"" + descriptorData + "\" > " + descriptorPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to write the cluster manifests")
	}

	ctx.Status.End(true) // End Generating worker cluster manifests

	ctx.Status.Start("Generating secrets file 📝🗝️")
	defer ctx.Status.End(false)

	commons.EnsureSecretsFile(a.keosCluster.Spec, a.vaultPassword, a.clusterCredentials)

	commons.RewriteDescriptorFile(a.descriptorPath)

	defer ctx.Status.End(true) // End Generating secrets file

	// Create namespace for CAPI clusters (it must exists)
	c = "kubectl create ns " + capiClustersNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster's Namespace")
	}

	// Create the allow-all-egress network policy file in the container
	allowCommonEgressNetPolPath := "/kind/allow-all-egress_netpol.yaml"
	c = "echo \"" + allowCommonEgressNetPol + "\" > " + allowCommonEgressNetPolPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to write the allow-all-egress network policy")
	}

	ctx.Status.Start("Installing keos cluster operator 💻")
	defer ctx.Status.End(false)

	err = deployClusterOperator(n, a.keosCluster, a.clusterCredentials, keosRegistry)
	if err != nil {
		return errors.Wrap(err, "failed to deploy cluster operator")
	}

	defer ctx.Status.End(true) // End installing keos cluster operator

	if !a.avoidCreation {
		if a.keosCluster.Spec.InfraProvider == "aws" && a.keosCluster.Spec.Security.AWS.CreateIAM {
			ctx.Status.Start("[CAPA] Ensuring IAM security 👮")
			defer ctx.Status.End(false)

			err = createCloudFormationStack(n, provider.capxEnvVars)
			if err != nil {
				return errors.Wrap(err, "failed to create the IAM security")
			}
			ctx.Status.End(true)
		}

		ctx.Status.Start("Creating the workload cluster 💥")
		defer ctx.Status.End(false)

		// Apply cluster manifests
		//c = "kubectl apply -n " + capiClustersNamespace + " -f " + descriptorPath
		c = "kubectl apply -f " + manifestsPath + "/keoscluster.yaml"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply manifests")
		}

		time.Sleep(10 * time.Second)

		// Wait for the control plane initialization
		c = "kubectl -n " + capiClustersNamespace + " wait --for=condition=ControlPlaneInitialized --timeout=25m cluster " + a.keosCluster.Metadata.Name
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create the workload cluster")
		}

		ctx.Status.End(true) // End Creating the workload cluster

		ctx.Status.Start("Saving the workload cluster kubeconfig 📝")
		defer ctx.Status.End(false)

		// Get the workload cluster kubeconfig
		c = "clusterctl -n " + capiClustersNamespace + " get kubeconfig " + a.keosCluster.Metadata.Name + " | tee " + kubeconfigPath
		kubeconfig, err := commons.ExecuteCommand(n, c)
		if err != nil || kubeconfig == "" {
			return errors.Wrap(err, "failed to get workload cluster kubeconfig")
		}

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
		if !a.keosCluster.Spec.ControlPlane.Managed {

			if a.keosCluster.Spec.InfraProvider == "azure" {
				ctx.Status.Start("Installing cloud-provider in workload cluster ☁️")
				defer ctx.Status.End(false)

				err = installCloudProvider(n, a.keosCluster, kubeconfigPath, a.keosCluster.Metadata.Name)
				if err != nil {
					return errors.Wrap(err, "failed to install external cloud-provider in workload cluster")
				}
				ctx.Status.End(true) // End Installing Calico in workload cluster
			}

			ctx.Status.Start("Installing Calico in workload cluster 🔌")
			defer ctx.Status.End(false)

			err = installCalico(n, kubeconfigPath, a.keosCluster, allowCommonEgressNetPolPath)
			if err != nil {
				return errors.Wrap(err, "failed to install Calico in workload cluster")
			}
			ctx.Status.End(true) // End Installing Calico in workload cluster

			ctx.Status.Start("Installing CSI in workload cluster 💾")
			defer ctx.Status.End(false)

			err = infra.installCSI(n, kubeconfigPath)
			if err != nil {
				return errors.Wrap(err, "failed to install CSI in workload cluster")
			}

			ctx.Status.End(true)
		}

		if provider.capxProvider == "gcp" {
			// XXX Ref kubernetes/kubernetes#86793 Starting from v1.18, gcp cloud-controller-manager requires RBAC to patch,update service/status (in-tree)
			ctx.Status.Start("Creating Kubernetes RBAC for internal loadbalancing 🔐")
			defer ctx.Status.End(false)

			requiredInternalNginx, err := infra.internalNginx(providerParams, a.keosCluster.Spec.Networks)
			if err != nil {
				return err
			}

			if requiredInternalNginx {
				rbacInternalLoadBalancingPath := "/kind/internalloadbalancing_rbac.yaml"

				// Deploy Kubernetes RBAC internal loadbalancing
				c = "echo \"" + rbacInternalLoadBalancing + "\" > " + rbacInternalLoadBalancingPath
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to write the kubernetes RBAC internal loadbalancing")
				}

				c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + rbacInternalLoadBalancingPath
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to the kubernetes RBAC internal loadbalancing")
				}
			}

			ctx.Status.End(true)
		}

		ctx.Status.Start("Preparing nodes in workload cluster 📦")
		defer ctx.Status.End(false)

		if awsEKSEnabled {
			c = "kubectl -n capa-system rollout restart deployment capa-controller-manager"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to reload capa-controller-manager")
			}
		}

		if provider.capxProvider != "azure" || !a.keosCluster.Spec.ControlPlane.Managed {
			// Wait for all the machine deployments to be ready
			c = "kubectl -n " + capiClustersNamespace + " wait --for=condition=Ready --timeout=15m --all md"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
		}

		if !a.keosCluster.Spec.ControlPlane.Managed && a.keosCluster.Spec.ControlPlane.HighlyAvailable {
			// Wait for all control planes to be ready
			c = "kubectl -n " + capiClustersNamespace + " wait --for=jsonpath=\"{.status.readyReplicas}\"=3 --timeout 10m kubeadmcontrolplanes " + a.keosCluster.Metadata.Name + "-control-plane"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
		}

		if azureAKSEnabled && a.keosCluster.Spec.Security.NodesIdentity != "" {
			// Update AKS cluster with the user kubelet identity until the provider supports it
			err := assignUserIdentity(providerParams, a.keosCluster.Spec.Security.NodesIdentity)
			if err != nil {
				return errors.Wrap(err, "failed to assign user identity to the workload Cluster")
			}
		}

		// Wait for metrics-server deployment to be ready
		if azureAKSEnabled {
			c = "kubectl --kubeconfig " + kubeconfigPath + " rollout status deploy metrics-server -n kube-system --timeout=5m"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
		}
		ctx.Status.End(true) // End Preparing nodes in workload cluster

		ctx.Status.Start("Installing StorageClass in workload cluster 💾")
		defer ctx.Status.End(false)

		err = infra.configureStorageClass(n, kubeconfigPath)
		if err != nil {
			return errors.Wrap(err, "failed to configure StorageClass in workload cluster")
		}
		ctx.Status.End(true) // End Installing StorageClass in workload cluster

		ctx.Status.Start("Enabling workload cluster's self-healing 🏥")
		defer ctx.Status.End(false)

		err = enableSelfHealing(n, a.keosCluster, capiClustersNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to enable workload cluster's self-healing")
		}

		ctx.Status.End(true) // End Enabling workload cluster's self-healing

		ctx.Status.Start("Installing CAPx in workload cluster 🎖️")
		defer ctx.Status.End(false)

		err = provider.installCAPXWorker(n, kubeconfigPath, allowCommonEgressNetPolPath)
		if err != nil {
			return err
		}

		// Scale CAPI to 2 replicas
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n capi-system scale --replicas 2 deploy capi-controller-manager"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to scale the CAPI Deployment")
		}

		// Allow egress in CAPI's Namespaces
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n capi-system apply -f " + allowCommonEgressNetPolPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply CAPI's egress NetworkPolicy")
		}
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n capi-kubeadm-bootstrap-system apply -f " + allowCommonEgressNetPolPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply CAPI's egress NetworkPolicy")
		}
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n capi-kubeadm-control-plane-system apply -f " + allowCommonEgressNetPolPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply CAPI's egress NetworkPolicy")
		}

		// Allow egress in cert-manager Namespace
		c = "kubectl --kubeconfig " + kubeconfigPath + " -n cert-manager apply -f " + allowCommonEgressNetPolPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply cert-manager's NetworkPolicy")
		}

		ctx.Status.End(true) // End Installing CAPx in workload cluster

		// Use Calico as network policy engine in managed systems
		if provider.capxProvider != "azure" {
			ctx.Status.Start("Configuring Network Policy Engine in workload cluster 🚧")
			defer ctx.Status.End(false)

			// Use Calico as network policy engine in managed systems
			if a.keosCluster.Spec.ControlPlane.Managed {

				err = installCalico(n, kubeconfigPath, a.keosCluster, allowCommonEgressNetPolPath)
				if err != nil {
					return errors.Wrap(err, "failed to install Network Policy Engine in workload cluster")
				}
			}

			// Create the allow and deny (global) network policy file in the container
			denyallEgressIMDSGNetPolPath := "/kind/deny-all-egress-imds_gnetpol.yaml"
			allowCAPXEgressIMDSGNetPolPath := "/kind/allow-egress-imds_gnetpol.yaml"

			// Allow egress in kube-system Namespace
			c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system apply -f " + allowCommonEgressNetPolPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to apply kube-system egress NetworkPolicy")
			}
			denyEgressIMDSGNetPol, err := provider.getDenyAllEgressIMDSGNetPol()
			if err != nil {
				return err
			}

			c = "echo \"" + denyEgressIMDSGNetPol + "\" > " + denyallEgressIMDSGNetPolPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to write the deny-all-traffic-to-aws-imds global network policy")
			}
			allowEgressIMDSGNetPol, err := provider.getAllowCAPXEgressIMDSGNetPol()
			if err != nil {
				return err
			}

			c = "echo \"" + allowEgressIMDSGNetPol + "\" > " + allowCAPXEgressIMDSGNetPolPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to write the allow-traffic-to-aws-imds-capa global network policy")
			}

			// Deny CAPA egress to AWS IMDS
			c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + denyallEgressIMDSGNetPolPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to apply deny IMDS traffic GlobalNetworkPolicy")
			}

			// Allow CAPA egress to AWS IMDS
			c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + allowCAPXEgressIMDSGNetPolPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to apply allow CAPX as egress GlobalNetworkPolicy")
			}

			ctx.Status.End(true) // End Installing Network Policy Engine in workload cluster
		}

		if a.keosCluster.Spec.DeployAutoscaler && !azureAKSEnabled {
			ctx.Status.Start("Adding Cluster-Autoescaler 🗚")
			defer ctx.Status.End(false)

			c = "helm install cluster-autoscaler /stratio/helm/cluster-autoscaler" +
				" --kubeconfig " + kubeconfigPath +
				" --namespace kube-system" +
				" --set autoDiscovery.clusterName=" + a.keosCluster.Metadata.Name +
				" --set autoDiscovery.labels[0].namespace=cluster-" + a.keosCluster.Metadata.Name +
				" --set cloudProvider=clusterapi" +
				" --set clusterAPIMode=incluster-incluster" +
				" --set replicaCount=2"

			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to install chart cluster-autoscaler")
			}

			ctx.Status.End(true)
		}

		// Apply custom CoreDNS configuration
		if a.keosCluster.Spec.Dns.Forwarders != nil && len(a.keosCluster.Spec.Dns.Forwarders) > 0 && !awsEKSEnabled {
			ctx.Status.Start("Customizing CoreDNS configuration 🪡")
			defer ctx.Status.End(false)

			err = customCoreDNS(n, kubeconfigPath, a.keosCluster)
			if err != nil {
				return errors.Wrap(err, "failed to customized CoreDNS configuration")
			}

			ctx.Status.End(true) // End Customizing CoreDNS configuration
		}

		// Create cloud-provisioner Objects backup
		ctx.Status.Start("Creating cloud-provisioner Objects backup 🗄️")
		defer ctx.Status.End(false)

		if _, err := os.Stat(localBackupPath); os.IsNotExist(err) {
			if err := os.MkdirAll(localBackupPath, 0755); err != nil {
				return errors.Wrap(err, "failed to create local backup directory")
			}
		}

		c = "mkdir -p " + cloudProviderBackupPath + " && chmod -R 0755 " + cloudProviderBackupPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create cloud-provisioner backup directory")
		}

		c = "clusterctl move -n " + capiClustersNamespace + " --to-directory " + cloudProviderBackupPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to backup cloud-provisioner Objects")
		}

		for _, path := range PathsToBackupLocally {
			raw := bytes.Buffer{}
			cmd := exec.CommandContext(context.Background(), "sh", "-c", "docker cp "+n.String()+":"+path+" "+localBackupPath)
			if err := cmd.SetStdout(&raw).Run(); err != nil {
				return errors.Wrap(err, "failed to copy "+path+" to local host")
			}
		}

		ctx.Status.End(true)

		if !a.moveManagement {
			ctx.Status.Start("Moving the management role 🗝️")
			defer ctx.Status.End(false)

			// Create namespace for CAPI clusters (it must exists) in worker cluster
			c = "kubectl --kubeconfig " + kubeconfigPath + " create ns " + capiClustersNamespace
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create manifests Namespace")
			}

			// Pivot management role to worker cluster
			c = "clusterctl move -n " + capiClustersNamespace + " --to-kubeconfig " + kubeconfigPath
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to pivot management role to worker cluster")
			}

			ctx.Status.End(true) // End Moving the management role

			ctx.Status.Start("Moving the cluster-operator 🗝️")

			// Create the docker registries credentials secret for keoscluster-controller-manager
			if a.clusterCredentials.DockerRegistriesCredentials != nil {
				c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system create secret generic keoscluster-registries --from-literal=credentials='" + string(jsonDockerRegistriesCredentials) + "'"
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to create keoscluster-registries secret")
				}
			}

			// Deploy cluster-operator chart in workload cluster
			c = "helm install cluster-operator /stratio/helm/cluster-operator" +
				" --kubeconfig " + kubeconfigPath +
				" --namespace kube-system" +
				" --set app.containers.controllerManager.image.registry=" + keosRegistry.url +
				" --set app.containers.controllerManager.image.repository=stratio/cluster-operator" +
				" --set app.containers.controllerManager.image.tag=" + keosClusterVersion
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to deploy cluster-operator chart in workload cluster")
			}

			// Wait for keoscluster-controller-manager deployment
			c = "kubectl --kubekubeconfig " + kubeconfigPath + " -n kube-system rollout status deploy/keoscluster-controller-manager --timeout=3m"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to wait for keoscluster-controller-manager deployment in workload cluster")
			}

			time.Sleep(10 * time.Second)

			//
			c = "kubectl -n " + capiClustersNamespace + " get keoscluster " + a.keosCluster.Metadata.Name + " -o yaml | kubectl apply --kubeconfig " + kubeconfigPath + " -f-"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to move keoscluster to workload cluster")
			}

			// Delete keoscluster in management cluster
			c = "kubectl -n " + capiClustersNamespace + " delete keoscluster " + a.keosCluster.Metadata.Name
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to delete keoscluster in management cluster")
			}

			ctx.Status.End(true) // End Moving the cluster-operator
		}
	}

	ctx.Status.Start("Generating the KEOS descriptor 📝")
	defer ctx.Status.End(false)

	err = createKEOSDescriptor(a.keosCluster, scName)
	if err != nil {
		return err
	}

	err = override_vars(ctx, providerParams, a.keosCluster.Spec.Networks, infra)
	if err != nil {
		return err
	}

	ctx.Status.End(true) // End Generating KEOS descriptor

	return nil
}
