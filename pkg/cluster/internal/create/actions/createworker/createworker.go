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

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

type action struct {
	vaultPassword  string
	descriptorPath string
	moveManagement bool
	avoidCreation  bool
}

const (
	kubeconfigPath          = "/kind/worker-cluster.kubeconfig"
	workKubeconfigPath      = ".kube/config"
	CAPILocalRepository     = "/root/.cluster-api/local-repository"
	cloudProviderBackupPath = "/kind/backup/objects"
	localBackupPath         = "backup"
)

var PathsToBackupLocally = []string{
	cloudProviderBackupPath,
	"/kind/manifests",
}

//go:embed files/all/allow-all-egress_netpol.yaml
var allowCommonEgressNetPol string

//go:embed files/gcp/rbac-loadbalancing.yaml
var rbacInternalLoadBalancing string

// In common with keos installer
//
//go:embed files/aws/deny-all-egress-imds_gnetpol.yaml
var denyallEgressIMDSGNetPol string

//go:embed files/aws/allow-capa-egress-imds_gnetpol.yaml
var allowCAPAEgressIMDSGNetPol string

// NewAction returns a new action for installing default CAPI
func NewAction(vaultPassword string, descriptorPath string, moveManagement bool, avoidCreation bool) actions.Action {
	return &action{
		vaultPassword:  vaultPassword,
		descriptorPath: descriptorPath,
		moveManagement: moveManagement,
		avoidCreation:  avoidCreation,
	}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	var c string
	var err error

	// Get the target node
	n, err := ctx.GetNode()
	if err != nil {
		return err
	}

	// Parse the cluster descriptor
	descriptorFile, err := commons.GetClusterDescriptor(a.descriptorPath)
	if err != nil {
		return errors.Wrap(err, "failed to parse cluster descriptor")
	}

	// Get the secrets

	credentialsMap, keosRegistry, githubToken, dockerRegistries, err := commons.GetSecrets(*descriptorFile, a.vaultPassword)
	if err != nil {
		return err
	}

	providerParams := commons.ProviderParams{
		Region:      descriptorFile.Region,
		Managed:     descriptorFile.ControlPlane.Managed,
		Credentials: credentialsMap,
		GithubToken: githubToken,
	}

	providerBuilder := getBuilder(descriptorFile.InfraProvider)
	infra := newInfra(providerBuilder)
	provider := infra.buildProvider(providerParams)

	ctx.Status.Start("Installing CAPx 🎖️")
	defer ctx.Status.End(false)

	if provider.capxVersion != provider.capxImageVersion {
		var registryUrl string
		var registryType string
		var registryUser string
		var registryPass string

		for _, registry := range descriptorFile.DockerRegistries {
			if registry.KeosRegistry {
				registryUrl = registry.URL
				registryType = registry.Type
				continue
			}
		}

		if registryType == "ecr" {
			ecrToken, err := getEcrToken(providerParams)
			if err != nil {
				return errors.Wrap(err, "failed to get ECR auth token")
			}
			registryUser = "AWS"
			registryPass = ecrToken
		} else if registryType == "acr" {
			acrService := strings.Split(registryUrl, "/")[0]
			acrToken, err := getAcrToken(providerParams, acrService)
			if err != nil {
				return errors.Wrap(err, "failed to get ACR auth token")
			}
			registryUser = "00000000-0000-0000-0000-000000000000"
			registryPass = acrToken
		} else {
			registryUser = keosRegistry["User"]
			registryPass = keosRegistry["Pass"]
		}

		// Change image in infrastructure-components.yaml
		infraComponents := CAPILocalRepository + "/infrastructure-" + provider.capxProvider + "/" + provider.capxVersion + "/infrastructure-components.yaml"
		infraImage := registryUrl + "/stratio/cluster-api-provider-" + provider.capxProvider + ":" + provider.capxImageVersion
		c = "sed -i 's%image:.*%image: " + infraImage + "%' " + infraComponents
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to change image in infrastructure-components.yaml")
		}

		// Create provider-system namespace
		c = "kubectl create namespace " + provider.capxName + "-system"
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create "+provider.capxName+"-system namespace")
		}

		// Create docker-registry secret
		c = "kubectl create secret docker-registry regcred" +
			" --docker-server=" + registryUrl +
			" --docker-username=" + registryUser +
			" --docker-password=" + registryPass +
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

	capiClustersNamespace := "cluster-" + descriptorFile.ClusterID

	templateParams := commons.TemplateParams{
		Descriptor:       *descriptorFile,
		Credentials:      credentialsMap,
		DockerRegistries: dockerRegistries,
	}

	azs, err := infra.getAzs(descriptorFile.Networks)
	if err != nil {
		return errors.Wrap(err, "failed to get AZs")
	}
	// Generate the cluster manifest

	descriptorData, err := GetClusterManifest(provider.capxTemplate, templateParams, azs)
	if err != nil {
		return errors.Wrap(err, "failed to generate cluster manifests")
	}

	// Create the cluster manifests file in the container
	descriptorPath := "/kind/manifests/cluster_" + descriptorFile.ClusterID + ".yaml"
	c = "echo \"" + descriptorData + "\" > " + descriptorPath
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to write the cluster manifests")
	}

	ctx.Status.End(true) // End Generating worker cluster manifests

	ctx.Status.Start("Generating secrets file 📝🗝️")
	defer ctx.Status.End(false)

	commons.EnsureSecretsFile(*descriptorFile, a.vaultPassword)

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

	if !a.avoidCreation {
		if descriptorFile.InfraProvider == "aws" && descriptorFile.Security.AWS.CreateIAM {
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
		c = "kubectl apply -n " + capiClustersNamespace + " -f " + descriptorPath
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to apply manifests")
		}

		// Wait for the control plane initialization
		c = "kubectl -n " + capiClustersNamespace + " wait --for=condition=ControlPlaneInitialized --timeout=25m cluster " + descriptorFile.ClusterID
		_, err = commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to create the worker Cluster")
		}

		ctx.Status.End(true) // End Creating the workload cluster

		ctx.Status.Start("Saving the workload cluster kubeconfig 📝")
		defer ctx.Status.End(false)

		// Get the workload cluster kubeconfig
		c = "clusterctl -n " + capiClustersNamespace + " get kubeconfig " + descriptorFile.ClusterID + " | tee " + kubeconfigPath
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
		if !descriptorFile.ControlPlane.Managed {

			if descriptorFile.InfraProvider == "azure" {
				ctx.Status.Start("Installing cloud-provider in workload cluster ☁️")
				defer ctx.Status.End(false)

				err = installCloudProvider(n, *descriptorFile, kubeconfigPath, descriptorFile.ClusterID)
				if err != nil {
					return errors.Wrap(err, "failed to install external cloud-provider in workload cluster")
				}
				ctx.Status.End(true) // End Installing Calico in workload cluster
			}

			ctx.Status.Start("Installing Calico in workload cluster 🔌")
			defer ctx.Status.End(false)

			err = installCalico(n, kubeconfigPath, *descriptorFile, allowCommonEgressNetPolPath)
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

		ctx.Status.Start("Installing StorageClass in workload cluster 💾")
		defer ctx.Status.End(false)

		err = infra.configureStorageClass(n, kubeconfigPath, descriptorFile.StorageClass)
		if err != nil {
			return errors.Wrap(err, "failed to configuring StorageClass in workload cluster")
		}
		ctx.Status.End(true) // End Installing StorageClass in workload cluster

		if provider.capxProvider == "gcp" {
			// XXX Ref kubernetes/kubernetes#86793 Starting from v1.18, gcp cloud-controller-manager requires RBAC to patch,update service/status (in-tree)
			ctx.Status.Start("Creating Kubernetes RBAC for internal loadbalancing 🔐")
			defer ctx.Status.End(false)

			requiredInternalNginx, err := infra.internalNginx(descriptorFile.Networks, credentialsMap, descriptorFile.ClusterID)
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

		if provider.capxProvider == "aws" && descriptorFile.ControlPlane.Managed {
			c = "kubectl -n capa-system rollout restart deployment capa-controller-manager"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to reload capa-controller-manager")
			}
		}

		if provider.capxProvider != "azure" || !descriptorFile.ControlPlane.Managed {
			// Wait for the worker cluster creation
			c = "kubectl -n " + capiClustersNamespace + " wait --for=condition=ready --timeout=15m --all md"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
		}

		if !descriptorFile.ControlPlane.Managed && descriptorFile.ControlPlane.HighlyAvailable {
			// Wait for all control planes creation
			c = "kubectl -n " + capiClustersNamespace + " wait --for=condition=ControlPlaneReady --timeout 10m cluster " + descriptorFile.ClusterID
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
			// Wait for all control planes to be ready
			c = "kubectl -n " + capiClustersNamespace + " wait --for=jsonpath=\"{.status.unavailableReplicas}\"=0 --timeout 10m --all kubeadmcontrolplanes"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to create the worker Cluster")
			}
		}

		if provider.capxProvider == "azure" && descriptorFile.ControlPlane.Managed && descriptorFile.Security.NodesIdentity != "" {
			// Update AKS cluster with the user kubelet identity until the provider supports it
			err := assignUserIdentity(descriptorFile.Security.NodesIdentity, descriptorFile.ClusterID, descriptorFile.Region, credentialsMap)
			if err != nil {
				return errors.Wrap(err, "failed to assign user identity to the workload Cluster")
			}
		}

		ctx.Status.End(true) // End Preparing nodes in workload cluster

		ctx.Status.Start("Enabling workload cluster's self-healing 🏥")
		defer ctx.Status.End(false)

		err = enableSelfHealing(n, *descriptorFile, capiClustersNamespace)
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
		if provider.capxProvider != "azure" && descriptorFile.ControlPlane.Managed {
			ctx.Status.Start("Installing Network Policy Engine in workload cluster 🚧")
			defer ctx.Status.End(false)

			err = installCalico(n, kubeconfigPath, *descriptorFile, allowCommonEgressNetPolPath)
			if err != nil {
				return errors.Wrap(err, "failed to install Network Policy Engine in workload cluster")
			}

			// Create the allow and deny (global) network policy file in the container
			if descriptorFile.InfraProvider == "aws" {
				denyallEgressIMDSGNetPolPath := "/kind/deny-all-egress-imds_gnetpol.yaml"
				allowCAPAEgressIMDSGNetPolPath := "/kind/allow-capa-egress-imds_gnetpol.yaml"

				// Allow egress in kube-system Namespace
				c = "kubectl --kubeconfig " + kubeconfigPath + " -n kube-system apply -f " + allowCommonEgressNetPolPath
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to apply kube-system egress NetworkPolicy")
				}

				c = "echo \"" + denyallEgressIMDSGNetPol + "\" > " + denyallEgressIMDSGNetPolPath
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to write the deny-all-traffic-to-aws-imds global network policy")
				}
				c = "echo \"" + allowCAPAEgressIMDSGNetPol + "\" > " + allowCAPAEgressIMDSGNetPolPath
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
				c = "kubectl --kubeconfig " + kubeconfigPath + " apply -f " + allowCAPAEgressIMDSGNetPolPath
				_, err = commons.ExecuteCommand(n, c)
				if err != nil {
					return errors.Wrap(err, "failed to apply allow CAPA as egress GlobalNetworkPolicy")
				}
			}
		}

		ctx.Status.End(true) // End Installing Network Policy Engine in workload cluster

		if descriptorFile.DeployAutoscaler && !(descriptorFile.InfraProvider == "azure" && descriptorFile.ControlPlane.Managed) {
			ctx.Status.Start("Adding Cluster-Autoescaler 🗚")
			defer ctx.Status.End(false)

			c = "helm install cluster-autoscaler /stratio/helm/cluster-autoscaler" +
				" --kubeconfig " + kubeconfigPath +
				" --namespace kube-system" +
				" --set autoDiscovery.clusterName=" + descriptorFile.ClusterID +
				" --set autoDiscovery.labels[0].namespace=cluster-" + descriptorFile.ClusterID +
				" --set cloudProvider=clusterapi" +
				" --set clusterAPIMode=incluster-incluster"

			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to install chart cluster-autoscaler")
			}

			ctx.Status.End(true)
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

			ctx.Status.End(true)
		}

	}

	ctx.Status.Start("Generating the KEOS descriptor 📝")
	defer ctx.Status.End(false)

	err = createKEOSDescriptor(*descriptorFile, provider.stClassName)
	if err != nil {
		return err
	}
	ctx.Status.End(true) // End Generating KEOS descriptor

	err = override_vars(*descriptorFile, credentialsMap, ctx, infra)
	if err != nil {
		return err
	}

	return nil
}
