#!/usr/bin/env python3
# -*- coding: utf-8 -*-

##############################################################
# Author: Stratio Clouds <clouds-integration@stratio.com>    #
# Supported provisioner versions: 0.7.X                      #
# Supported cloud providers:                                 #
#   - EKS                                                    #
#   - Azure VMs                                              #
#   - GKE                                                    #
##############################################################

__version__ = "0.8.0"

import argparse
import os
import sys
import json
import subprocess
import yaml
import base64
import logging
import re
import zlib
import time
from datetime import datetime
from ansible_vault import Vault
from jinja2 import Template, Environment, FileSystemLoader
from ruamel.yaml import YAML
from io import StringIO
from urllib.parse import urlparse

CLOUD_PROVISIONER = "0.17.0-0.8"
CLUSTER_OPERATOR = "0.7.0-m.1"
CLUSTER_OPERATOR_UPGRADE_SUPPORT = "0.5.X"
CLOUD_PROVISIONER_LAST_PREVIOUS_RELEASE = "0.17.0-0.7"

AWS_LOAD_BALANCER_CONTROLLER_CHART = "1.11.0"

CLUSTERCTL = "v1.10.8"

CAPI = "v1.10.8"
CAPI_KUBEADM_BOOTSTRAP = "v1.10.8"
CAPI_KUBEADM_CONTROL_PLANE = "v1.10.8"
CAPA = "v2.9.2"
CAPG = "1.6.1-0.4.0"
CAPZ = "v1.21.1"

TIGERA_OPERATOR_CALICOCTL_VERSION = "3.30.2"
TIGERA_OPERATOR_CONTROLLER_VERSION = "v1.38.5"

common_charts = {
    "cert-manager": {
        "version": "v1.19.1",
        "namespace": "cert-manager",
        "repo": "https://charts.jetstack.io"
    },
    "cluster-autoscaler": {
        "version": "9.52.1",
        "namespace": "kube-system",
        "repo": "https://kubernetes.github.io/autoscaler"
    },
    "cluster-operator": {
        "version": "0.7.0-m.1",
        "namespace": "kube-system",
        "repo": ""
    },
    "flux2": {
        "version": "2.17.2",
        "namespace": "kube-system",
        "repo": "https://fluxcd-community.github.io/helm-charts",
        "release_name": "flux"
    },
    "tigera-operator": {
        "version": "v3.30.2",
        "namespace": "tigera-operator",
        "repo": "https://docs.projectcalico.org/charts"
    }
}

aws_eks_charts = {
    "aws-load-balancer-controller": {
        "version": "1.14.1",
        "namespace": "kube-system",
        "repo": "https://aws.github.io/eks-charts"
    }
}

azure_vm_charts = {
    "azuredisk-csi-driver": {
        "version": "1.33.5",
        "namespace": "kube-system",
        "repo": "https://raw.githubusercontent.com/kubernetes-sigs/azuredisk-csi-driver/master/charts"
    },
    "azurefile-csi-driver": {
        "version": "1.34.1",
        "namespace": "kube-system",
        "repo": "https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/charts"
    },
    "cloud-provider-azure": {
        "version": "1.34.2",
        "namespace": "kube-system",
        "repo": "https://raw.githubusercontent.com/kubernetes-sigs/cloud-provider-azure/master/helm/repo"
    }
}

# List of GCP (CAPG) CRD names that require conversion webhook cleanup before clusterctl upgrade
CAPG_CRDS = [
    "gcpclusters.infrastructure.cluster.x-k8s.io",
    "gcpclustertemplates.infrastructure.cluster.x-k8s.io",
    "gcpmanagedclusters.infrastructure.cluster.x-k8s.io",
    "gcpmanagedcontrolplanes.infrastructure.cluster.x-k8s.io",
    "gcpmanagedmachinepools.infrastructure.cluster.x-k8s.io",
    "gcpmachines.infrastructure.cluster.x-k8s.io",
    "gcpmachinetemplates.infrastructure.cluster.x-k8s.io",
]

def patch_capg_crds_live():
    for crd in CAPG_CRDS:
        # Remove conversion webhook from CRD to avoid caBundle PEM errors during clusterctl upgrade.
        # Both patches are best-effort: if the conversion block doesn't exist the patch is a no-op.
        print(f"[INFO] Removing conversion webhook from {crd} (best-effort):", end=" ", flush=True)
        run_command(
            f"{kubectl} patch crd {crd} --type=json "
            "-p='[{\"op\":\"remove\",\"path\":\"/spec/conversion\"}]'",
            allow_errors=True
        )
        run_command(
            f"{kubectl} patch crd {crd} --type=merge "
            "-p='{\"spec\":{\"conversion\":{\"strategy\":\"None\"}}}'",
            allow_errors=True
        )
        print("OK (best-effort)")

# Set up Jinja2 environment to load templates from the templates directory
template_dir = './templates'
env = Environment(loader=FileSystemLoader(template_dir))

# Load Jinja2 templates for HelmRepository and HelmRelease manifests
helmrepository_template = env.get_template('helmrepository_template.yaml')
helmrelease_template = env.get_template('helmrelease_template.yaml')

def parse_args():
    parser = argparse.ArgumentParser(
        description='''This script upgrades cloud-provisioner from ''' + CLOUD_PROVISIONER_LAST_PREVIOUS_RELEASE + ''' to ''' + CLOUD_PROVISIONER +
                    ''' by upgrading mainly cluster-operator from ''' + CLUSTER_OPERATOR_UPGRADE_SUPPORT + ''' to ''' + CLUSTER_OPERATOR + ''' .
                        It requires kubectl, helm and jq binaries in $PATH.
                        A component (or all) must be selected for upgrading.
                        By default, the process will wait for confirmation for every component selected for upgrade.''',
                                    formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument("-y", "--yes", action="store_true", help="Do not wait for confirmation between tasks")
    parser.add_argument("-k", "--kubeconfig", help="Set the kubeconfig file for kubectl commands, It can also be set using $KUBECONFIG variable", default="~/.kube/config")
    parser.add_argument("-p", "--vault-password", help="Set the vault password for decrypting secrets", required=True)
    parser.add_argument("-s", "--secrets", help="Set the secrets file for decrypting secrets", default="secrets.yml")
    parser.add_argument("--cluster-operator", help="Set the cluster-operator target version", default=CLUSTER_OPERATOR)
    parser.add_argument("--disable-backup", action="store_true", help="Disable backing up files before upgrading (enabled by default)")
    parser.add_argument("--disable-prepare-capsule", action="store_true", help="Disable preparing capsule for the upgrade process (enabled by default)")
    parser.add_argument("--dry-run", action="store_true", help="Do not upgrade components. This invalidates all other options")
    parser.add_argument("--private", action="store_true", help="Treats the Docker registry and the Helm repository as private")
    parser.add_argument("--ecr-pull-through", action="store_true", help="Force ECR pull-through cache mode regardless of KeosCluster spec")
    args = parser.parse_args()
    return vars(args)

def backup(backup_dir, namespace, cluster_name, dry_run):
    '''Backup CAPX cluster move files, capsule webhooks and CAPI/CAPX namespace secrets'''

    print("[INFO] Backing up files into directory " + backup_dir)
    # Backup CAPX files
    print("[INFO] Backing up CAPX files:", end =" ", flush=True)
    if dry_run:
        print("DRY-RUN")
    else:
        os.makedirs(backup_dir + "/" + namespace, exist_ok=True)
        command = "clusterctl --kubeconfig " + kubeconfig + " -n cluster-" + cluster_name + " move --to-directory " + backup_dir + "/" + namespace + " >/dev/null 2>&1"
        status, output = subprocess.getstatusoutput(command)
        if status != 0:
            print("FAILED")
            print("[ERROR] Backing up CAPX files failed:\n" + output)
            sys.exit(1)
        else:
            print("OK")
    # Backup capsule files
    print("[INFO] Backing up capsule files:", end =" ", flush=True)
    if not dry_run:
        os.makedirs(backup_dir + "/capsule", exist_ok=True)
        capsule_backed_up = False
        command = kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration"
        status, _ = subprocess.getstatusoutput(command)
        if status == 0:
            command = kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration -o yaml 2>/dev/null > " + backup_dir + "/capsule/capsule-mutating-webhook-configuration.yaml"
            status, output = subprocess.getstatusoutput(command)
            if status != 0:
                print("FAILED")
                print("[ERROR] Backing up capsule files failed:\n" + output)
                sys.exit(1)
            capsule_backed_up = True
        command = kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration"
        status, output = subprocess.getstatusoutput(command)
        if status == 0:
            command = kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration -o yaml 2>/dev/null > " + backup_dir + "/capsule/capsule-validating-webhook-configuration.yaml"
            status, output = subprocess.getstatusoutput(command)
            if status != 0:
                print("FAILED")
                print("[ERROR] Backing up capsule files failed:\n" + output)
                sys.exit(1)
            capsule_backed_up = True
        if capsule_backed_up:
            print("OK")
        else:
            print("SKIP")
    else:
        print("DRY-RUN")
    # Backup CAPI/CAPA/CAPZ/CAPG secrets
    capx_namespaces = [
        "capi-system",
        "capi-kubeadm-bootstrap-system",
        "capi-kubeadm-control-plane-system",
        "capa-system",
        "capz-system",
        "capg-system",
    ]
    print("[INFO] Backing up CAPX secrets:", end=" ", flush=True)
    if dry_run:
        print("DRY-RUN")
    else:
        capx_secrets_backed_up = False
        for ns in capx_namespaces:
            # Check if the namespace exists
            check_ns_cmd = kubectl + f" get namespace {ns} --ignore-not-found -o name"
            ns_status, ns_output = subprocess.getstatusoutput(check_ns_cmd)
            if ns_status != 0 or not ns_output.strip():
                continue
            # List secrets in the namespace
            list_cmd = kubectl + f" get secret -n {ns} -o name"
            list_status, list_output = subprocess.getstatusoutput(list_cmd)
            if list_status != 0 or not list_output.strip():
                continue
            ns_backup_dir = backup_dir + "/capx-secrets/" + ns
            os.makedirs(ns_backup_dir, exist_ok=True)
            for secret_ref in list_output.strip().splitlines():
                secret_name = secret_ref.split("/")[-1]
                out_file = ns_backup_dir + "/" + secret_name + ".yaml"
                dump_cmd = kubectl + f" get secret -n {ns} {secret_name} -o yaml 2>/dev/null > {out_file}"
                dump_status, dump_output = subprocess.getstatusoutput(dump_cmd)
                if dump_status != 0:
                    print("FAILED")
                    print(f"[ERROR] Backing up secret {ns}/{secret_name} failed:\n" + dump_output)
                    sys.exit(1)
                capx_secrets_backed_up = True
        if capx_secrets_backed_up:
            print("OK")
        else:
            print("SKIP")

def prepare_capsule(dry_run):
    '''Prepare capsule for the upgrade process'''

    print("[INFO] Preparing capsule-mutating-webhook-configuration for the upgrade process:", end =" ", flush=True)
    if not dry_run:
        command = kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration"
        status, output = subprocess.getstatusoutput(command)
        if status != 0:
            if "NotFound" in output:
                print("SKIP")
            else:
                print("FAILED")
                print("[ERROR] Preparing capsule-mutating-webhook-configuration failed:\n" + output)
                sys.exit(1)
        else:
            command = (kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration -o json | " +
                    '''jq -r '.webhooks[0].objectSelector |= {"matchExpressions":[{"key":"name","operator":"NotIn","values":["kube-system","tigera-operator","calico-system","cert-manager","capi-system","''' +
                    namespace + '''","capi-kubeadm-bootstrap-system","capi-kubeadm-control-plane-system"]},{"key":"kubernetes.io/metadata.name","operator":"NotIn","values":["kube-system","tigera-operator","calico-system","cert-manager","capi-system","''' +
                    namespace + '''","capi-kubeadm-bootstrap-system","capi-kubeadm-control-plane-system"]}]}' | ''' + kubectl + " apply -f -")
            execute_command(command, False)
    else:
        print("DRY-RUN")

    print("[INFO] Preparing capsule-validating-webhook-configuration for the upgrade process:", end =" ", flush=True)
    if not dry_run:
        command = kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration"
        status, output = subprocess.getstatusoutput(command)
        if status != 0:
            if "NotFound" in output:
                print("SKIP")
            else:
                print("FAILED")
                print("[ERROR] Preparing capsule-validating-webhook-configuration failed:\n" + output)
                sys.exit(1)
        else:
            command = (kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration -o json | " +
                    '''jq -r '.webhooks[] |= (select(.name == "namespaces.capsule.clastix.io").objectSelector |= ({"matchExpressions":[{"key":"name","operator":"NotIn","values":["''' +
                    namespace + '''","tigera-operator","calico-system"]},{"key":"kubernetes.io/metadata.name","operator":"NotIn","values":["''' +
                    namespace + '''","tigera-operator","calico-system"]}]}))' | ''' + kubectl + " apply -f -")
            execute_command(command, False)
    else:
        print("DRY-RUN")

def restore_capsule(dry_run):
    '''Restore capsule after the upgrade process'''

    print("[INFO] Restoring capsule-mutating-webhook-configuration:", end =" ", flush=True)
    if not dry_run:
        command = kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration"
        status, output = subprocess.getstatusoutput(command)
        if status != 0:
            if "NotFound" in output:
                print("SKIP")
            else:
                print("FAILED")
                print("[ERROR] Restoring capsule-mutating-webhook-configuration failed:\n" + output)
                sys.exit(1)
        else:
            command = (kubectl + " get mutatingwebhookconfigurations capsule-mutating-webhook-configuration -o json | " +
                    "jq -r '.webhooks[0].objectSelector |= {}' | " + kubectl + " apply -f -")
            execute_command(command, False)
    else:
        print("DRY-RUN")

    print("[INFO] Restoring capsule-validating-webhook-configuration:", end =" ", flush=True)
    if not dry_run:
        command = kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration"
        status, output = subprocess.getstatusoutput(command)
        if status != 0:
            if "NotFound" in output:
                print("SKIP")
            else:
                print("FAILED")
                print("[ERROR] Restoring capsule-validating-webhook-configuration failed:\n" + output)
                sys.exit(1)
        else:
            command = (kubectl + " get validatingwebhookconfigurations capsule-validating-webhook-configuration -o json | " +
                    """jq -r '.webhooks[] |= (select(.name == "namespaces.capsule.clastix.io").objectSelector |= {})' """ +
                    "| " + kubectl + " apply -f -")
            execute_command(command, False)
    else:
        print("DRY-RUN")

def patch_clusterrole_aws_node(dry_run):
    '''Patch aws-node ClusterRole'''

    aws_node_clusterrole_name = "aws-node"
    print("[INFO] Modifying aws-node ClusterRole:", end =" ", flush=True)
    if not dry_run:
        command = f"{kubectl} get clusterrole -o json {aws_node_clusterrole_name} | jq -r '.rules'"
        cluster_role_rules_output = execute_command(command, False, False)

        try:
            cluster_role_rules = json.loads(cluster_role_rules_output)
        except json.JSONDecodeError as e:
            print(f"[ERROR] Failed to parse ClusterRole rules as JSON: {e}")
            sys.exit(1)

        rule_pods_index = next((i for i, rule in enumerate(cluster_role_rules) if 'pods' in rule.get('resources', [])), None)
        if rule_pods_index is not None:
            verbs = cluster_role_rules[rule_pods_index].get('verbs', [])
            if 'patch' not in verbs:
                patch = [
                    {
                        "op": "add",
                        "path": f"/rules/{rule_pods_index}/verbs/-",
                        "value": "patch"
                    }
                ]
                patch_command = f"{kubectl} patch clusterrole {aws_node_clusterrole_name} --type=json -p='{json.dumps(patch)}'"
                execute_command(patch_command, False, True)
            else:
                print("SKIP")
        else:
            print(f"[ERROR] Pods resource not found in the ClusterRole {aws_node_clusterrole_name}")
            sys.exit(1)
    else:
        print("DRY-RUN")

def scale_cluster_autoscaler(replicas, dry_run):
    '''Scale cluster-autoscaler deployment'''

    command = kubectl + " get deploy cluster-autoscaler-clusterapi-cluster-autoscaler -n kube-system --ignore-not-found -o=jsonpath='{.spec.replicas}'"
    output = execute_command(command, False, False)

    if output.strip() == "":
        print("[INFO] Cluster autoscaler not deployed: SKIP")
        return

    current_replicas = int(output)

    if current_replicas == replicas:
        print("[INFO] Cluster autoscaler already at desired replicas: SKIP")
        return

    scaling_type = "Scaling down" if current_replicas > replicas else "Scaling up"
    print(f"[INFO] {scaling_type} cluster autoscaler replicas:", end=" ", flush=True)

    if dry_run:
        print("DRY-RUN")
        return

    # Scale
    command = kubectl + f" scale deploy cluster-autoscaler-clusterapi-cluster-autoscaler -n kube-system --replicas={replicas}"
    execute_command(command, False, False)

    # Wait until ready
    command = kubectl + " wait deployment cluster-autoscaler-clusterapi-cluster-autoscaler -n kube-system --for=condition=Available --timeout=5m"
    execute_command(command, False, False)

    print("OK")

def wait_for_keos_cluster(cluster_name, timeout_minutes):
    '''Wait for the KeosCluster to be ready'''

    command = (
        "kubectl wait --for=jsonpath=\"{.status.ready}\"=true KeosCluster "
        + cluster_name + " -n cluster-" + cluster_name + " --timeout "+timeout_minutes+"m"
    )
    execute_command(command, False, False)

def validate_helm_repository(helm_repository):
    '''Validate the Helm repository'''

    try:
        url = urlparse(helm_repository)
        if not all([url.scheme, url.netloc]):
            raise ValueError(f"The Helm repository '{helm_repository}' is invalid.")
    except ValueError:
        raise ValueError(f"The Helm repository '{helm_repository}' is invalid.")

def update_helm_repository(cluster_name, helm_repository, dry_run):
    '''Update the Helm repository'''

    wait_for_keos_cluster(cluster_name, "10")


    patch_helm_repository = [
        {"op": "replace", "path": "/spec/helm_repository/url", "value": helm_repository},
    ]

    patch_json = json.dumps(patch_helm_repository)
    command = f"{kubectl} -n cluster-{cluster_name} patch KeosCluster {cluster_name} --type='json' -p='{patch_json}'"
    execute_command(command, False, False)

    patch_helmRepository = [
        {"op": "replace", "path": "/spec/url", "value": helm_repository},
    ]
    patch_json = json.dumps(patch_helmRepository)
    existing_helmrepo, err = run_command(f"{kubectl} get helmrepository -n kube-system keos --ignore-not-found", allow_errors=True)
    if "doesn't have a resource type \"helmrepository\"" in err:
        existing_helmrepo = False

    if existing_helmrepo:
        command = f"{kubectl} -n kube-system patch helmrepository keos --type='json' -p='{patch_json}'"
        execute_command(command, False, False)

    wait_for_keos_cluster(cluster_name, "10")

def execute_command(command, dry_run, result = True, max_retries=3, retry_delay=5):
    '''Execute a command and handle the output'''

    output = ""
    retries = 0

    while retries < max_retries:
        if dry_run:
            if result:
                print("DRY-RUN")
            return ""  # No output in dry-run mode
        else:
            status, output = subprocess.getstatusoutput(command)
            if status == 0:
                if result:
                    print("OK")
                return output
            else:
                retries += 1
                if retries < max_retries:
                    time.sleep(retry_delay)
                else:
                    print("FAILED")
                    print("[ERROR] " + output)
                    sys.exit(1)

def get_chart_version(chart, namespace):
    '''Get the version of a Helm chart'''

    command = helm + " -n " + namespace + " list"
    output = execute_command(command, False, False)
    for line in output.split("\n"):
        splitted_line = line.split()
        if chart == splitted_line[0]:
            # helm list output columns (0-indexed):
            # 0:NAME  1:NAMESPACE  2:REVISION  3:UPDATED(date)  4:UPDATED(time)
            # 5:UPDATED(timezone)  6:UPDATED(utc)  7:STATUS  8:CHART  9:APP_VERSION
            if chart == "cluster-operator":
                return splitted_line[9]
            else:
                return splitted_line[8].split("-")[-1]
    return None

def get_version(version):
    '''Get the version number'''

    return re.sub(r'\D', '', version)

def print_upgrade_support():
    '''Print the upgrade support message'''

    print("[WARN] Upgrading cloud-provisioner from a version minor than " + CLOUD_PROVISIONER_LAST_PREVIOUS_RELEASE + " to " + CLOUD_PROVISIONER + " is NOT SUPPORTED")
    print("[WARN] You have to upgrade to cloud-provisioner:"+ CLOUD_PROVISIONER_LAST_PREVIOUS_RELEASE + " first")
    sys.exit(0)

def request_confirmation():
    '''Request confirmation to continue'''

    enter = input("Press ENTER to continue upgrading the cluster or any other key to abort: ")
    if enter != "":
        sys.exit(0)

def get_keos_cluster_cluster_config():
    '''Get the KeosCluster and ClusterConfig objects'''

    try:
        keoscluster_list_output, err = run_command(kubectl + " get keoscluster -A -o json")
        keos_cluster = json.loads(keoscluster_list_output)["items"][0]
        clusterconfig_list_output, err = run_command(kubectl + " get clusterconfig -A -o json")
        cluster_config = json.loads(clusterconfig_list_output)["items"][0]
        return keos_cluster, cluster_config
    except Exception as e:
        print(f"[ERROR] {e}.")
        raise e


def run_command(command, allow_errors=False, retries=3, retry_delay=2):

    if config["dry_run"]:
        mutating_keywords = [
            " apply ",
            " patch ",
            " delete ",
            " scale ",
            " create ",
            " annotate ",
            " label ",
            " upgrade apply ",
        ]

        normalized_command = f" {command.lower()} "

        if any(keyword in normalized_command for keyword in mutating_keywords):
            print("[DRY-RUN] Skipping mutating command")
            return "", ""

    attempts = 0

    while attempts <= retries:
        result = subprocess.run(command, shell=True, capture_output=True, text=True)

        if result.returncode == 0:
            return result.stdout, result.stderr

        # If the command fails and the error is allowed, return the result without raising an exception
        if allow_errors:
            return result.stdout, result.stderr

        # If the command fails and the error is not allowed, but there are retries left, wait and retry
        attempts += 1
        if attempts > retries:
            raise Exception(f"Error executing '{command}': {result.stderr}")

        time.sleep(retry_delay)

def get_helm_repository(keos_cluster):
    '''Get the Helm registry URL'''

    try:
        helm_repository = keos_cluster["spec"]["helm_repository"]["url"]

        if helm_repository:
            return helm_repository
        else:
            return None
    except KeyError as e:
        return None

def get_deploy_version(deploy, namespace, container):
    '''Get the version of a deployment'''

    command = f"{kubectl} -n " + namespace + " get deploy " + deploy + " -o json  | jq -r '.spec.template.spec.containers[].image' | grep '" + container + "' | cut -d: -f2"
    output = execute_command(command, False, False)
    return output.split("@")[0]

def update_annotation_label(annotation_label_key, annotation_label_value, resources, type="annotation"):
    '''Update the annotation or label of a resource'''

    for resource in resources:
        kind = resource["kind"]
        name = resource["name"]
        ns = resource.get("namespace")
        action_type = "annotate"
        if type == "label":
            action_type = "label"
        try:
            command = f"{kubectl} get {kind} {name} "
            if ns:
                command = command + f" -n {ns}"
            output, err = run_command(command, allow_errors=True)
            if "not found" in err.lower():

                continue
        except Exception as e:
            print("FAILED")
            print(f"[ERROR] Error checking the existence of {kind} {name}: {e}")
            return

        command = f"{kubectl} {action_type} {kind} {name} {annotation_label_key}={annotation_label_value} --overwrite "
        if ns:
            command = command + f" -n {ns}"
        output, err = run_command(command)

def get_keos_registry_url(keos_cluster):
    '''Get the Keos registry URL'''

    docker_registries = keos_cluster["spec"]["docker_registries"]
    for registry in docker_registries:
        if registry.get("keos_registry", False):
            return registry["url"]
    return ""

def is_ecr_pull_through_enabled(keos_cluster):
    '''Return True if ECR pull-through cache is enabled in the cluster or forced via --ecr-pull-through flag'''

    if config.get("ecr_pull_through", False):
        return True
    for registry in keos_cluster["spec"].get("docker_registries", []):
        if registry.get("ecr_pull_through_cache_enabled", False):
            return True
    return False

def get_pods_cidr(keos_cluster):
    '''Get the pods CIDR'''

    try:
        return keos_cluster["spec"]["networks"]["pods_cidr"]
    except KeyError:
        return ""

def is_private_registry_enabled(cluster_config):
    '''Return the effective private registry setting'''

    return cluster_config.get("spec", {}).get("private_registry", False) or config.get("private", False)

def is_private_helm_repo_enabled(cluster_config):
    '''Return the effective private Helm repository setting'''
    return cluster_config.get("spec", {}).get("private_helm_repo", False) or config.get("private", False)

def render_values_template(values_file, keos_cluster, cluster_config):
    '''Render the values template'''

    try:
        values_params = {
            "private": is_private_registry_enabled(cluster_config),
            "cluster_name": keos_cluster["metadata"]["name"],
            "registry": get_keos_registry_url(keos_cluster),
            "provider": keos_cluster["spec"]["infra_provider"],
            "managed_cluster": keos_cluster["spec"]["control_plane"]["managed"]
        }

        template = env.get_template(values_file)
        rendered_values = template.render(values_params)
        return rendered_values
    except Exception as e:
        raise e

def create_default_values(chart_name, namespace, values_file, provider):
    '''Create defaults values file'''

    charts_requiring_values_update_all = []
    charts_requiring_values_update_provider = []
    try:
        if chart_name in charts_requiring_values_update_all:
            values = render_values_template( f"values/{chart_name}_default_values.tmpl", keos_cluster, cluster_config)
        elif chart_name in charts_requiring_values_update_provider:
            values = render_values_template( f"values/{provider}/{chart_name}_default_values.tmpl", keos_cluster, cluster_config)
        else:
            values, err = run_command(f"{helm} get values {chart_name} -n {namespace} --output yaml")
        if is_ecr_pull_through_enabled(keos_cluster):
            registry_url = get_keos_registry_url(keos_cluster)
            pull_through_substitutions = [
                (f"{registry_url}/tigera",    f"{registry_url}/quay/tigera"),
                (f"{registry_url}/jetstack",  f"{registry_url}/quay/jetstack"),
                (f"{registry_url}/fluxcd",    f"{registry_url}/ghcr/fluxcd"),
                (f"{registry_url}/autoscaling", f"{registry_url}/k8s/autoscaling"),
                (f"{registry_url}/eks/",      f"{registry_url}/ecrpublic/eks/"),
            ]
            for old, new in pull_through_substitutions:
                values = values.replace(old, new)
        run_command(f"echo '{values}' > {values_file}")
    except Exception as e:
        raise

def update_cluster_operator_image_tag_value(values_file, cluster_operator_version):
    '''Update cluster-operator image tag value'''

    try:
        with open(values_file, 'r') as file:
            values = yaml.safe_load(file)

        values['app']['containers']['controllerManager']['image']['tag'] = cluster_operator_version

        with open(values_file, 'w') as file:
            yaml.safe_dump(values, file, default_flow_style=False)

    except Exception as e:
        print(f"An error occurred: {e}")

def update_tigera_operator_image_tag_value(values_file):
    '''Update tigera-operator calicoctl and controller image tag values'''

    try:
        with open(values_file, 'r') as file:
            values = yaml.safe_load(file)

        values['calicoctl']['tag'] = TIGERA_OPERATOR_CALICOCTL_VERSION
        values['tigeraOperator']['version'] = TIGERA_OPERATOR_CONTROLLER_VERSION

        # Apply ECR pull-through prefixes to registry fields.
        # The registry URL is stored separately from the image path in tigera values,
        # so string substitution in create_default_values() never matches — must be done here.
        if is_ecr_pull_through_enabled(keos_cluster):
            registry_url = get_keos_registry_url(keos_cluster)
            quay_registry = f"{registry_url}/quay"
            dockerhub_registry = f"{registry_url}/dockerhub"
            if values.get('tigeraOperator', {}).get('registry', '').startswith(registry_url) and \
               not values['tigeraOperator']['registry'].startswith(quay_registry):
                values['tigeraOperator']['registry'] = quay_registry
            if 'installation' in values:
                values['installation']['registry'] = quay_registry
                values['installation']['imagePath'] = 'calico'
            calico_image = values.get('calicoctl', {}).get('image', '')
            if calico_image.startswith(registry_url) and not calico_image.startswith(dockerhub_registry):
                values['calicoctl']['image'] = calico_image.replace(
                    f"{registry_url}/calico", f"{dockerhub_registry}/calico", 1)

        with open(values_file, 'w') as file:
            yaml.safe_dump(values, file, default_flow_style=False)

    except Exception as e:
        print(f"An error occurred: {e}")

def create_empty_values_file(values_file):
    ''' Create an empty values file'''

    try:
        open(values_file, 'w').close()
    except Exception as e:
        raise e

def create_configmap_from_values(configmap_name, namespace, values_file):
    '''Create a ConfigMap from values'''

    try:
        command = f"{kubectl} create configmap {configmap_name} -n {namespace} --from-file=values.yaml={values_file} --dry-run=client -o yaml | kubectl apply -f -"
        run_command(command)
    except Exception as e:
        raise e

def filter_installed_charts(charts):
    '''Remove not installed charts'''

    try:
        output, err = run_command(helm  + " list --all-namespaces --output json")
        charts_installed = json.loads(output)
        charts_installed_names = [chart["name"] for chart in charts_installed]

        charts_filtered = {
            chart_name: chart_data
            for chart_name, chart_data in charts.items()
            if chart_data.get("release_name", chart_name) in charts_installed_names
        }
        return charts_filtered
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error getting charts installed {e}.")
        raise e

def apply_chart_crds(chart_name, chart_version, repo_url, repo_schema):
    '''Pull chart and apply CRDs — Helm upgrade never updates CRDs, must be done explicitly'''

    import tempfile
    import glob

    print(f"[INFO] Applying CRDs for {chart_name} {chart_version}:", end=" ", flush=True)
    try:
        with tempfile.TemporaryDirectory() as tmpdir:
            if repo_schema == "oci":
                registry = repo_url.replace("oci://", "").split("/")[0]
                if ".dkr.ecr." in registry:
                    region = registry.split(".")[3]
                    run_command(f"aws ecr get-login-password --region {region} | {helm} registry login {registry} --username AWS --password-stdin")
                pull_cmd = f"{helm} pull {repo_url}/{chart_name} --version {chart_version} -d {tmpdir}"
            else:
                pull_cmd = f"{helm} pull {chart_name} --repo {repo_url} --version {chart_version} -d {tmpdir}"
            run_command(pull_cmd)

            tarballs = glob.glob(f"{tmpdir}/*.tgz")
            if not tarballs:
                print("SKIP (no tarball found)")
                return

            tarball = tarballs[0]
            run_command(f"tar xzf {tarball} -C {tmpdir} {chart_name}/crds/ 2>/dev/null || true")

            crd_files = glob.glob(f"{tmpdir}/{chart_name}/crds/*.yaml")
            if not crd_files:
                print("SKIP (no CRDs in chart)")
                return

            for crd_file in crd_files:
                run_command(f"{kubectl} apply -f {crd_file}")

        print("OK")
    except Exception as e:
        print(f"WARN ({e}) — continuing without CRD update")


def upgrade_chart(chart_name, chart_data):
    '''Update chart HelmRelease'''
    chart_repo = chart_data["repo"]
    chart_version = chart_data["version"]
    chart_namespace = chart_data["namespace"]

    release_name = chart_name
    if chart_name == "flux2":
        release_name = "flux"
    repo_name = release_name
    repo_schema = "default"
    repo_username = ""
    repo_password = ""
    repo_auth_required = False
    repo_url = chart_repo

    if chart_name == "cluster-operator" or private_helm_repo:
        repo_name = "keos"
        repo_url =  keos_cluster["spec"]["helm_repository"]["url"]
        if "auth_required" in keos_cluster["spec"]["helm_repository"]:
            if keos_cluster["spec"]["helm_repository"]["auth_required"]:
                if "user" in vault_secrets_data["secrets"]["helm_repository"] and "pass" in vault_secrets_data["secrets"]["helm_repository"]:
                    repo_auth_required= True
                    repo_username = vault_secrets_data["secrets"]["helm_repository"]["user"]
                    repo_password = vault_secrets_data["secrets"]["helm_repository"]["pass"]
                else:
                    print("[ERROR] Helm repository credentials not found in secrets file")
                    sys.exit(1)
        if urlparse(repo_url).scheme == "oci":
            repo_schema = "oci"

    default_values_file = f"/tmp/{release_name}_default_values.yaml"
    empty_values_file = f"/tmp/{release_name}_empty_values.yaml"

    # Cleanup function for temp files
    def cleanup_temp_files():
        for temp_file in [default_values_file, empty_values_file]:
            if os.path.exists(temp_file):
                try:
                    os.remove(temp_file)
                except Exception:
                    pass

    try:
        create_default_values(release_name, chart_namespace, default_values_file, provider)
        if release_name == "cluster-operator":
            update_cluster_operator_image_tag_value(default_values_file, cluster_operator_version)
        elif release_name == "tigera-operator":
            update_tigera_operator_image_tag_value(default_values_file)

        create_empty_values_file(empty_values_file)

        create_configmap_from_values(f"00-{release_name}-helm-chart-default-values", chart_namespace, default_values_file)
        create_configmap_from_values(f"02-{release_name}-helm-chart-override-values", chart_namespace, empty_values_file)

        helm_repo_data = {
            'repository_name': repo_name,
            'namespace': chart_namespace,
            'interval': '10m',
            'repository_url': repo_url,
            'schema': repo_schema,
            'provider': provider,
            'auth_required': repo_auth_required,
            'username': repo_username,
            'password': repo_password
        }

        helm_release_data = {
            'ReleaseName': release_name,
            'ChartName': chart_name,
            'ChartNamespace': chart_namespace,
            'ChartVersion': chart_version,
            'ChartRepoRef': repo_name,
            'HelmReleaseSourceInterval': '1m',
            'HelmReleaseInterval': '1m',
            'HelmReleaseRetries': 3
        }

        if chart_name == "cluster-operator":
            apply_chart_crds(chart_name, chart_version, repo_url, repo_schema)

        helmrepository_yaml = helmrepository_template.render(helm_repo_data)
        helmrelease_yaml = helmrelease_template.render(helm_release_data)

        repository_file = f'/tmp/{release_name}_helmrepository.yaml'
        release_file = f'/tmp/{release_name}_helmrelease.yaml'

        with open(repository_file, 'w') as f:
            f.write(helmrepository_yaml)

        with open(release_file, 'w') as f:
            f.write(helmrelease_yaml)

        # We need to use --server-side and --force-conflicts flags to avoid metadata.resourceVersion conflicts
        run_command(f"{kubectl} apply -f {repository_file} --server-side --force-conflicts")
        run_command(f"{kubectl} apply -f {release_file} -n {chart_namespace} --server-side --force-conflicts")

        print("OK")

        # Cleanup temp files after successful apply
        cleanup_temp_files()
        if os.path.exists(repository_file):
            os.remove(repository_file)
        if os.path.exists(release_file):
            os.remove(release_file)

    except Exception as e:
        cleanup_temp_files()
        raise e

def upgrade_charts(charts):
    '''Update the charts'''

    try:
        print(f"[INFO] Updating charts versions:")
        for chart_name, chart_data in charts.items():
            chart_version = chart_data["version"]
            print(f"[INFO] Updating chart {chart_name} to version {chart_version}:", end =" ", flush=True)
            upgrade_chart(chart_name, chart_data)
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error updating chart: {e}")
        raise e

def stop_keoscluster_controller():
    '''Stop the KEOSCluster controller'''

    try:
        print("[INFO] Stopping keoscluster-controller-manager deployment:", end =" ", flush=True)
        run_command(f"{kubectl} scale deployment -n kube-system keoscluster-controller-manager --replicas=0", allow_errors=True)

        print("OK")
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error stopping the KEOSCluster controller: {e}")
        raise e

def disable_keoscluster_webhooks():
    '''Disable the KEOSCluster webhooks'''

    try:
        backup_keoscluster_webhooks()
        print("[INFO] Disabling KEOSCluster webhooks:", end =" ", flush=True)

        run_command(f"{kubectl} delete validatingwebhookconfiguration keoscluster-validating-webhook-configuration", allow_errors=True)
        run_command(f"{kubectl} delete mutatingwebhookconfiguration keoscluster-mutating-webhook-configuration", allow_errors=True)
        print("OK")
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error disabling KEOSCluster webhooks: {e}")
        raise e

def backup_keoscluster_webhooks():
    '''Backup the KEOSCluster webhooks'''

    backup_file = backup_dir + "/cluster-operator/keoscluster-webhooks.yaml"
    try:
        if not os.path.exists(os.path.dirname(backup_file)):
            os.makedirs(os.path.dirname(backup_file))
        print("[INFO] Backing up KEOSCluster webhook configurations:", end =" ", flush=True)
        command = f"{helm} get manifest -n kube-system cluster-operator"
        command += f" | yq 'select(.kind == \"ValidatingWebhookConfiguration\" or .kind == \"MutatingWebhookConfiguration\")'"
        command += f" > {backup_file}"
        execute_command(command, False)
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error backing up KEOSCluster webhooks: {e}")
        raise e

def update_clusterconfig(cluster_config, charts, provider, cluster_operator_version):
    '''Update the clusterconfig'''

    try:
        print("[INFO] Updating clusterconfig:", end =" ", flush=True)

        clusterconfig_name = cluster_config["metadata"]["name"]
        clusterconfig_namespace = cluster_config["metadata"]["namespace"]

        # ------------------------------------------------------------------
        # Update cluster-operator
        # ------------------------------------------------------------------
        cluster_config["spec"]["cluster_operator_version"] = cluster_operator_version
        cluster_config["spec"]["cluster_operator_image_version"] = cluster_operator_version
        cluster_config["spec"]["private_registry"] = is_private_registry_enabled(cluster_config)
        cluster_config["spec"]["private_helm_repo"] = is_private_helm_repo_enabled(cluster_config)

        # ------------------------------------------------------------------
        # Update CAPX (Cluster API providers)
        # ------------------------------------------------------------------
        if "capx" not in cluster_config["spec"]:
            cluster_config["spec"]["capx"] = {}

        # Always update CAPI
        cluster_config["spec"]["capx"]["capi_version"] = CAPI

        if provider == "aws":
            cluster_config["spec"]["capx"]["capa_version"] = CAPA
            cluster_config["spec"]["capx"]["capa_image_version"] = CAPA

        elif provider == "gcp":
            cluster_config["spec"]["capx"]["capg_version"] = CAPG
            cluster_config["spec"]["capx"]["capg_image_version"] = CAPG

        elif provider == "azure":
            cluster_config["spec"]["capx"]["capz_version"] = CAPZ
            cluster_config["spec"]["capx"]["capz_image_version"] = CAPZ

        # ------------------------------------------------------------------
        # Update Helm charts list
        # ------------------------------------------------------------------
        cluster_config["spec"]["charts"] = []
        for chart_name, chart_data in charts.items():
            cluster_config["spec"]["charts"].append({
                "name": chart_name,
                "version": chart_data["version"]
            })

        # ------------------------------------------------------------------
        # Patch ClusterConfig
        # ------------------------------------------------------------------
        clusterconfig_json = json.dumps(cluster_config)
        command = (
            f"{kubectl} patch clusterconfig {clusterconfig_name} "
            f"-n {clusterconfig_namespace} --type merge -p '{clusterconfig_json}'"
        )

        run_command(command)

        print("OK")

    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error updating the clusterconfig: {e}")
        raise e

def create_clusterctl_config_for_private_registry(registry_url, provider, pull_through=False):
    """Create or update clusterctl config file to use private registry"""
    print("[INFO] Configuring clusterctl for private registry:", end=" ", flush=True)

    config_dir = os.path.expanduser("~/.cluster-api")
    config_file = os.path.join(config_dir, "clusterctl.yaml")

    # Create config directory if it doesn't exist
    os.makedirs(config_dir, exist_ok=True)

    # Read existing config or create new one
    config_data = {}
    if os.path.exists(config_file):
        # Backup existing config
        backup_file = config_file + ".backup-" + datetime.now().strftime("%Y%m%d-%H%M%S")
        run_command(f"cp {config_file} {backup_file}", allow_errors=True)
        print(f"\n[DEBUG] Backed up existing config to {backup_file}")

        # Load existing config
        with open(config_file, 'r') as f:
            config_data = yaml.safe_load(f) or {}
        print(f"[DEBUG] Loaded existing clusterctl config")

    # Update images section for private registry
    if 'images' not in config_data:
        config_data['images'] = {}

    k8s_prefix  = "k8s/"  if pull_through else ""
    quay_prefix = "quay/" if pull_through else ""

    # Align the image overrides with the original installation logic.
    config_data['images']['cluster-api'] = {
        'repository': f"{registry_url}/{k8s_prefix}cluster-api",
        'tag': CAPI,
    }
    config_data['images']['bootstrap-kubeadm'] = {
        'repository': f"{registry_url}/{k8s_prefix}cluster-api",
        'tag': CAPI,
    }
    config_data['images']['control-plane-kubeadm'] = {
        'repository': f"{registry_url}/{k8s_prefix}cluster-api",
        'tag': CAPI,
    }
    config_data['images']['cert-manager'] = {
        'repository': f"{registry_url}/{quay_prefix}jetstack"
    }

    if provider == "aws":
        config_data['images']['infrastructure-aws'] = {
            'repository': f"{registry_url}/{k8s_prefix}cluster-api-aws",
            'tag': CAPA,
        }
    elif provider == "gcp":
        config_data['images']['infrastructure-gcp'] = {
            'repository': f"{registry_url}/stratio",
            'tag': CAPG,
        }
    elif provider == "azure":
        config_data['images']['infrastructure-azure/cluster-api-azure-controller'] = {
            'repository': f"{registry_url}/cluster-api-azure",
            'tag': CAPZ,
        }
        config_data['images']['infrastructure-azure/azureserviceoperator'] = {
            'repository': f"{registry_url}/k8s"
        }
        config_data['images']['infrastructure-azure/kube-rbac-proxy'] = {
            'repository': f"{registry_url}/kubebuilder"
        }
        config_data['images']['infrastructure-azure/nmi'] = {
            'repository': f"{registry_url}/oss/azure/aad-pod-identity"
        }

    # Write updated configuration
    with open(config_file, 'w') as f:
        yaml.safe_dump(config_data, f, default_flow_style=False, sort_keys=False)

    print("OK")
    print(f"[DEBUG] Updated clusterctl config at {config_file}")
    print(f"[DEBUG] Images will be pulled from private registry: {registry_url}")

    if provider == "gcp":
        # GCP local manifests need additional image rewrites beyond clusterctl overrides.
        patch_local_repository_manifests(config_dir, registry_url)

def patch_local_repository_manifests(config_dir, registry_url):
    """Patch local repository YAML manifests to use private registry"""
    print("[INFO] Patching local repository manifests:", end=" ", flush=True)

    local_repo = os.path.join(config_dir, "local-repository")
    if not os.path.exists(local_repo):
        print("SKIP (no local repository found)")
        return

    patched_count = 0
    total_replacements = 0

    for root, _, files in os.walk(local_repo):
        for file in files:
            if file.endswith(".yaml"):
                filepath = os.path.join(root, file)
                try:
                    with open(filepath, 'r') as f:
                        content = f.read()

                    # Replace registry.k8s.io with private registry
                    # This regex captures the full image path
                    original_content = content
                    new_content, count = re.subn(
                        r'registry\.k8s\.io/([^\s:"\']+)',
                        f'{registry_url}/stratio/\\1',
                        content
                    )
                    # Fix CAPG manifest that uses e2e image tag
                    new_content, gcp_count = re.subn(
                        r'gcr\.io/k8s-staging-cluster-api-gcp/cluster-api-gcp-controller:[^\s"\']+',
                        f'{registry_url}/stratio/cluster-api-gcp-controller:{CAPG}',
                        new_content
                    )
                    count += gcp_count
                    # Only write if changes were made
                    if new_content != original_content:
                        with open(filepath, 'w') as f:
                            f.write(new_content)
                        patched_count += 1
                        total_replacements += count
                        print(f"\n[DEBUG] Patched {filepath}: {count} replacements", flush=True)
                except Exception as e:
                    print(f"\n[WARN] Failed to patch {filepath}: {e}")

    print(f"\nOK ({patched_count} files patched, {total_replacements} total replacements)")

def patch_gcp_crd_conversion_webhook(config_dir):
    """Remove conversion webhook from GCP CRDs in local repository to avoid caBundle errors"""
    print("[INFO] Removing conversion webhooks from GCP CRDs:", end=" ", flush=True)

    gcp_repo = os.path.join(config_dir, "local-repository", "infrastructure-gcp")
    if not os.path.exists(gcp_repo):
        print("SKIP (no GCP repository found)")
        return

    patched_files = []
    patched_crds = 0

    for root, _, files in os.walk(gcp_repo):
        for file in files:
            if file.endswith(".yaml"):
                filepath = os.path.join(root, file)
                try:
                    with open(filepath, 'r') as f:
                        content = f.read()

                    yaml_parser = YAML()
                    yaml_parser.preserve_quotes = True
                    yaml_parser.width = 4096

                    documents = list(yaml_parser.load_all(content))
                    modified = False

                    for doc in documents:
                        if doc and doc.get('kind') == 'CustomResourceDefinition':
                            crd_name = doc.get('metadata', {}).get('name', '')
                            if crd_name in CAPG_CRDS:
                                if 'spec' in doc and 'conversion' in doc['spec']:
                                    del doc['spec']['conversion']
                                    modified = True
                                    patched_crds += 1
                                    print(f"\n[DEBUG] Removed conversion from {crd_name} in {filepath}", flush=True)

                        # Fallback: if conversion block still exists but empty/partial, force strategy None
                        if doc and doc.get('kind') == 'CustomResourceDefinition':
                            crd_name = doc.get('metadata', {}).get('name', '')
                            if crd_name in CAPG_CRDS:
                                if 'spec' in doc and 'conversion' in doc['spec']:
                                    doc['spec']['conversion'] = {"strategy": "None"}
                                    modified = True
                                    print(f"\n[DEBUG] Forced conversion.strategy=None for {crd_name} in {filepath}", flush=True)

                    if modified:
                        output = StringIO()
                        yaml_parser.dump_all(documents, output)
                        with open(filepath, 'w') as f:
                            f.write(output.getvalue())
                        patched_files.append(filepath)

                except Exception as e:
                    print(f"\n[WARN] Failed to patch CRD in {filepath}: {e}")

    if patched_files:
        print(f"\nOK ({len(patched_files)} files patched, {patched_crds} CRDs patched)")
    else:
        print("OK (no CRDs needed patching)")

def patch_clusterctl_images(registry_url):
    '''Patch Cluster API provider image references to use a private registry'''
    print("[INFO] Patching Cluster API provider images for private registry:", end=" ", flush=True)

    repo_base = os.environ.get("CAPI_REPO")
    if not repo_base:
        print("SKIP (CAPI_REPO not set)")
        return

    for root, _, files in os.walk(repo_base):
        for file in files:
            if file.endswith(".yaml"):
                filepath = os.path.join(root, file)

                with open(filepath, "r") as f:
                    content = f.read()

                content = re.sub(
                    r"registry\.k8s\.io",
                    registry_url,
                    content
                )

                with open(filepath, "w") as f:
                    f.write(content)

    print("OK")

def upgrade_cluster_api_providers(provider):
    '''Upgrade Cluster API core and infrastructure providers using clusterctl'''
    print("[INFO] Upgrading Cluster API providers:", end=" ", flush=True)

    command = (
        f"{env_vars} clusterctl upgrade apply "
        f"--kubeconfig {kubeconfig} "
        f"--core cluster-api:{CAPI} "
    )

    # Bootstrap and control-plane providers are only needed for unmanaged clusters (Azure VMs).
    # EKS and GKE manage the control plane themselves, so these providers are not upgraded.
    if provider == "azure":
        command += (
            f"--bootstrap kubeadm:{CAPI_KUBEADM_BOOTSTRAP} "
            f"--control-plane kubeadm:{CAPI_KUBEADM_CONTROL_PLANE} "
        )

    if provider == "aws":
        command += f"--infrastructure aws:{CAPA} "
    elif provider == "azure":
        command += f"--infrastructure azure:{CAPZ} "
    elif provider == "gcp":
        command += f"--infrastructure gcp:{CAPG} "

    command += "--wait-providers"

    safe_command = re.sub(r"GCP_B64ENCODED_CREDENTIALS=\S+", "GCP_B64ENCODED_CREDENTIALS=<redacted>", command)
    print(f"\n[DEBUG] Full clusterctl command: {safe_command}", flush=True)

    run_command(command)

    print("OK")

def restore_keoscluster_webhooks():
    '''Restore the KEOSCluster webhooks'''

    backup_file = backup_dir + "/cluster-operator/keoscluster-webhooks.yaml"
    resources_webhooks = [
        {"kind": "MutatingWebhookConfiguration", "name": "keoscluster-mutating-webhook-configuration", "namespace": "kube-system"},
        {"kind": "ValidatingWebhookConfiguration", "name": "keoscluster-validating-webhook-configuration", "namespace": "kube-system"},
    ]
    try:
        print("[INFO] Restoring KEOSCluster webhooks from backup:", end =" ", flush=True)
        run_command(f"{kubectl} create -f {backup_file}", allow_errors=True)
        print("OK")

        print("[INFO] Labeling and annotating webhooks:", end =" ", flush=True)
        update_annotation_label("app.kubernetes.io/managed-by", "Helm", resources_webhooks, "label")
        update_annotation_label("meta.helm.sh/release-name", "cluster-operator", resources_webhooks)
        update_annotation_label("meta.helm.sh/release-namespace", "kube-system", resources_webhooks)
        print("OK")
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error restoring KEOSCluster webhooks from backup: {e}")
        raise e

def start_keoscluster_controller():
    '''Start the KEOSCluster controller'''

    try:
        print("[INFO] Starting keoscluster-controller-manager deployment:", end =" ", flush=True)

        run_command(f"{kubectl} scale deployment -n kube-system keoscluster-controller-manager --replicas=2")
        run_command(f"{kubectl} wait --for=condition=Available deployment/keoscluster-controller-manager -n kube-system --timeout=300s")
        print("OK")

    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Error starting the KEOSCluster controller: {e}")

        raise e

def configure_aws_credentials(vault_secrets_data):
    print("[INFO] Configuring AWS CLI credentials", end=" ", flush=True)

    aws_creds = vault_secrets_data['secrets']['aws']['credentials']
    aws_access_key = aws_creds['access_key']
    aws_secret_key = aws_creds['secret_key']
    aws_region = aws_creds['region']
    role_arn = aws_creds.get('role_arn')

    # Disable AWS pager inside containers
    os.environ["AWS_PAGER"] = ""

    # Base credentials
    os.environ["AWS_ACCESS_KEY_ID"] = aws_access_key
    os.environ["AWS_SECRET_ACCESS_KEY"] = aws_secret_key
    os.environ["AWS_DEFAULT_REGION"] = aws_region

    # If role_arn exists → assume role
    if role_arn:
        assume_cmd = [
            "aws", "sts", "assume-role",
            "--role-arn", role_arn,
            "--role-session-name", "upgrade-session"
        ]

        result = subprocess.run(assume_cmd, capture_output=True, text=True)
        if result.returncode != 0:
            print("FAILED")
            print(result.stderr)
            sys.exit(1)

        creds = json.loads(result.stdout)["Credentials"]

        os.environ["AWS_ACCESS_KEY_ID"] = creds["AccessKeyId"]
        os.environ["AWS_SECRET_ACCESS_KEY"] = creds["SecretAccessKey"]
        os.environ["AWS_SESSION_TOKEN"] = creds["SessionToken"]

    print("OK")

def configure_azure_credentials(vault_secrets_data):
    print("[INFO] Configuring Azure CLI credentials", end=" ", flush=True)
    azure_client_id = vault_secrets_data['secrets']['azure']['credentials']['client_id']
    azure_client_secret = vault_secrets_data['secrets']['azure']['credentials']['client_secret']
    azure_subscription_id = vault_secrets_data['secrets']['azure']['credentials']['subscription_id']
    azure_tenant_id = vault_secrets_data['secrets']['azure']['credentials']['tenant_id']

    command = f"az login --service-principal --username {azure_client_id} \
                --password {azure_client_secret} --tenant {azure_tenant_id}"

    try:
        run_command(command)
        print("OK")
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] Azure CLI login failed: {e}")
        sys.exit(1)

def configure_gcp_credentials(vault_secrets_data):
    """Configure GCP gcloud credentials from service account key"""
    print("[INFO] Configuring GCP gcloud credentials", end=" ", flush=True)

    try:
        gcp_creds = vault_secrets_data['secrets']['gcp']['credentials']
        project_id = gcp_creds['project_id']

        # Check if service_account_key exists (JSON key content)
        if 'service_account_key' in gcp_creds:
            # Write service account key to temporary file
            import tempfile
            with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as key_file:
                json.dump(gcp_creds['service_account_key'], key_file)
                key_path = key_file.name

            # Activate service account
            activate_cmd = f"gcloud auth activate-service-account --key-file={key_path} --quiet"
            result = subprocess.run(activate_cmd, shell=True, capture_output=True, text=True)

            # Clean up key file
            os.remove(key_path)

            if result.returncode != 0:
                print("FAILED")
                print(f"[ERROR] gcloud auth failed: {result.stderr}")
                sys.exit(1)

        # Set default project
        project_cmd = f"gcloud config set project {project_id} --quiet"
        result = subprocess.run(project_cmd, shell=True, capture_output=True, text=True)

        if result.returncode != 0:
            print("FAILED")
            print(f"[ERROR] Setting GCP project failed: {result.stderr}")
            sys.exit(1)

        print("OK")

    except KeyError as e:
        print("FAILED")
        print(f"[ERROR] Missing GCP credential field: {e}")
        sys.exit(1)
    except Exception as e:
        print("FAILED")
        print(f"[ERROR] GCP credential configuration failed: {e}")
        sys.exit(1)

def activate_capg_service_account(kubectl, kubeconfig):
    """
    Activates the CAPG service account from capg-manager-bootstrap-credentials
    and regenerates kubeconfig so gke-gcloud-auth-plugin uses this identity.
    """

    print("[INFO] Activating CAPG service account:", end=" ", flush=True)

    try:
        # Get credentials.json from secret
        cmd = (
            f"{kubectl} -n capg-system get secret "
            f"capg-manager-bootstrap-credentials "
            f"-o jsonpath='{{.data.credentials\\.json}}'"
        )

        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)

        if result.returncode != 0 or not result.stdout.strip():
            print("FAILED")
            print(result.stderr)
            sys.exit(1)

        credentials_json = base64.b64decode(result.stdout.strip()).decode("utf-8")
        credentials = json.loads(credentials_json)

        # Write temporary key file
        import tempfile
        key_file = tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False)
        key_file.write(credentials_json)
        key_file.close()
        key_path = key_file.name

        # Activate SA
        subprocess.run(
            f"gcloud auth activate-service-account "
            f"--key-file={key_path} --quiet",
            shell=True,
            check=True
        )

        # Set project
        subprocess.run(
            f"gcloud config set project {credentials['project_id']} --quiet",
            shell=True,
            check=True
        )

        # Extract cluster name from kubeconfig context
        context = subprocess.check_output(
            f"kubectl --kubeconfig {kubeconfig} config current-context",
            shell=True,
            text=True
        ).strip()

        cluster_name = context.split("_")[-1]

        # 🔥 Get region from VAULT secrets (NOT from CAPG secret)
        gcp_creds = vault_secrets_data['secrets']['gcp']['credentials']
        location = gcp_creds.get('region') or gcp_creds.get('zone')

        if not location:
            raise Exception("GCP region/zone not found in vault secrets")

        # Refresh kubeconfig using correct region
        subprocess.run(
            f"KUBECONFIG={kubeconfig} "
            f"gcloud container clusters get-credentials {cluster_name} "
            f"--region {location} "
            f"--project {credentials['project_id']}",
            shell=True,
            check=True
        )

        print("OK")

    except Exception as e:
        print("FAILED")
        print(e)
        sys.exit(1)

if __name__ == '__main__':
    # Set start time
    start_time = time.time()
    print("[INFO] Starting cluster upgrade process")
    print("[INFO] Setting up the environment...")

    # Set backup directory
    backup_dir = "./backup/upgrade/"
    print("[INFO] Backup directory: " + backup_dir)

    # Configure the logger
    logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
    logger = logging.getLogger(__name__)

    # Parse arguments
    config = parse_args()

    # Set kubeconfig
    print("[INFO] Setting kubeconfig:", end =" ", flush=True)
    if os.environ.get("KUBECONFIG"):
        kubeconfig = os.environ.get("KUBECONFIG")
    else:
        kubeconfig = os.path.expanduser(config["kubeconfig"])
    print("OK")

    # Check clusterctl version
    print("[INFO] Checking clusterctl version:", end =" ", flush=True)
    command = "clusterctl version -o short"
    status, output = subprocess.getstatusoutput(command)
    if (status != 0) or (get_version(output) < get_version(CLUSTERCTL)):
        print("[ERROR] clusterctl version " + CLUSTERCTL + " is required")
        sys.exit(1)
    print("OK")

    # Check if secrets file and kubeconfig file exist
    print("[INFO] Checking secrets file and kubeconfig file:", end =" ", flush=True)
    if not os.path.exists(config["secrets"]):
        print("[ERROR] Secrets file not found")
        sys.exit(1)
    if not os.path.exists(kubeconfig):
        print("[ERROR] Kubeconfig file not found")
        sys.exit(1)
    print("OK")

    # Get data from vault secrets file (secrets.yml)
    print("[INFO] Reading secrets file", end =" ", flush=True)
    try:
        vault = Vault(config["vault_password"])
        vault_secrets_data = vault.load(open(config["secrets"]).read())
    except Exception as e:
        print("[ERROR] Decoding secrets file failed:\n" + str(e))
        sys.exit(1)
    print("OK")

    # Configure cloud provider CLI
    if 'aws' in vault_secrets_data['secrets']:
        configure_aws_credentials(vault_secrets_data)
    elif 'azure' in vault_secrets_data['secrets']:
        configure_azure_credentials(vault_secrets_data)
    elif 'gcp' in vault_secrets_data['secrets']:
        configure_gcp_credentials(vault_secrets_data)
    else:
        print("[ERROR] Unable to detect provider from secrets file for CLI configuration")
        sys.exit(1)

    # Print kubeconfig path
    print("[INFO] Using kubeconfig: " + kubeconfig)

    # Set kubectl
    print("[INFO] Setting kubectl with kubeconfig", end =" ", flush=True)
    kubectl = "kubectl --kubeconfig " + kubeconfig
    print("OK")

    # Set helm
    print("[INFO] Setting helm with kubeconfig", end =" ", flush=True)
    helm = "helm --kubeconfig " + kubeconfig
    print("OK")

    # Detect provider early from secrets file
    if 'aws' in vault_secrets_data['secrets']:
        provider = "aws"
    elif 'azure' in vault_secrets_data['secrets']:
        provider = "azure"
    elif 'gcp' in vault_secrets_data['secrets']:
        provider = "gcp"
    else:
        print("[ERROR] Unable to detect provider from secrets file")
        sys.exit(1)

    print("[INFO] Detected provider: " + provider)

    # Extract cluster name from kubeconfig context
    try:
        context_cmd = f"kubectl --kubeconfig {kubeconfig} config current-context"
        current_context = subprocess.check_output(context_cmd, shell=True, text=True, stderr=subprocess.DEVNULL).strip()
        # Try to extract cluster name from context (format varies by provider)
        # For EKS: usually contains cluster name
        # For AKS: format is typically clustername
        # For GKE: format is gke_project_zone_clustername
        if provider == "gcp" and "gke_" in current_context:
            cluster_name_guess = current_context.split("_")[-1]
        elif provider == "aws" and "@" in current_context:
            cluster_name_guess = current_context.split("@")[1].split(".")[0]
        else:
            cluster_name_guess = current_context.split("/")[-1].split("@")[-1]
        print(f"[INFO] Detected cluster name from context: {cluster_name_guess}")
    except Exception as e:
        cluster_name_guess = None
        print(f"[WARN] Could not extract cluster name from kubeconfig context: {e}")

    # Validate kubectl access BEFORE trying to get resources
    print("[INFO] Validating kubectl access to the cluster:", end =" ", flush=True)

    def test_kubectl():
        command = kubectl + " get ns >/dev/null 2>&1"
        return subprocess.call(command, shell=True) == 0

    # If kubectl access fails, attempt kubeconfig refresh for supported providers before exiting with error
    if not test_kubectl():
        print("FAILED (attempting kubeconfig refresh)", flush=True)

        if provider == "aws":
            region = vault_secrets_data['secrets']['aws']['credentials']['region']

            # Try to get cluster name from context or use provided name
            if cluster_name_guess:
                cluster_name_for_refresh = cluster_name_guess
            else:
                print("[ERROR] Cannot refresh kubeconfig: cluster name not detected from context")
                print("[HINT] Ensure your kubeconfig has a valid context set")
                sys.exit(1)

            refresh_cmd = (
                f"aws eks update-kubeconfig "
                f"--name {cluster_name_for_refresh} "
                f"--region {region} "
                f"--kubeconfig {kubeconfig}"
            )

            print(f"[INFO] Attempting to refresh kubeconfig for cluster: {cluster_name_for_refresh}")
            status = subprocess.call(refresh_cmd, shell=True)

            if status != 0:
                print("[ERROR] Failed to refresh kubeconfig")
                sys.exit(1)

            # Rebuild kubectl with refreshed kubeconfig
            kubectl = "kubectl --kubeconfig " + kubeconfig

            if not test_kubectl():
                print("[ERROR] kubectl still failing after kubeconfig refresh")
                sys.exit(1)

            print("OK (kubeconfig refreshed)")

        elif provider == "azure":
            print("[ERROR] kubectl access failed")
            print("[HINT] For Azure, refresh upgrade credentials:")
            print("[HINT] For Azure, refresh credentials by updating the kubeconfig file:")
            print(f"  1. Locate your kubeconfig at: {kubeconfig}")
            print(f"  2. Update the authentication credentials manually in the file")
            print(f"  3. Ensure the user credentials or service principal tokens are valid")
            print("[ACTION REQUIRED] After updating the credentials in the kubeconfig file, please re-run this script")
            sys.exit(1)

        elif provider == "gcp":
            # Get GCP credentials from vault
            gcp_creds = vault_secrets_data['secrets']['gcp']['credentials']
            project_id = gcp_creds['project_id']
            region = gcp_creds.get('region', gcp_creds.get('zone'))  # Support both region and zone

            # Try to get cluster name from context or use provided name
            if cluster_name_guess:
                cluster_name_for_refresh = cluster_name_guess
            else:
                print("[ERROR] Cannot refresh kubeconfig: cluster name not detected from context")
                print("[HINT] Ensure your kubeconfig has a valid context set")
                sys.exit(1)

            # Determine if it's a regional or zonal cluster
            if region:
                location_flag = f"--region {region}"
            else:
                print("[ERROR] Cannot refresh kubeconfig: region/zone not found in secrets")
                print("[HINT] Ensure 'region' or 'zone' is set in GCP credentials")
                sys.exit(1)

            refresh_cmd = (
                f"KUBECONFIG={kubeconfig} gcloud container clusters get-credentials {cluster_name_for_refresh} "
                f"{location_flag} "
                f"--project {project_id}"
            )

            print(f"[INFO] Attempting to refresh kubeconfig for cluster: {cluster_name_for_refresh}")
            status = subprocess.call(refresh_cmd, shell=True)

            if status != 0:
                print("[ERROR] Failed to refresh kubeconfig")
                sys.exit(1)

            # Rebuild kubectl with refreshed kubeconfig
            kubectl = "kubectl --kubeconfig " + kubeconfig

            if not test_kubectl():
                print("[ERROR] kubectl still failing after kubeconfig refresh")
                sys.exit(1)

            print("OK (kubeconfig refreshed)")

        else:
            print("[ERROR] kubectl access failed and auto-refresh not supported for this provider")
            sys.exit(1)
    else:
        print("OK")

    # Activate CAPG service account and refresh GKE kubeconfig after kubectl validation
    if provider == "gcp":
        activate_capg_service_account(kubectl, kubeconfig)

    # Get KeosCluster and ClusterConfig
    print("[INFO] Getting KeosCluster and ClusterConfig", end =" ", flush=True)
    keos_cluster, cluster_config = get_keos_cluster_cluster_config()
    print("OK")

    # Get cluster_name from KeosCluster metadata
    print("[INFO] Getting cluster name from KeosCluster metadata", end =" ", flush=True)
    if "metadata" in keos_cluster:
        cluster_name = keos_cluster["metadata"]["name"]
    else:
        print("[ERROR] KeosCluster definition not found. Ensure that KeosCluster is defined before ClusterConfig in the descriptor file")
        sys.exit(1)
    print("OK")

    print("[INFO] Cluster name: " + cluster_name)

    # Verify provider matches
    provider_from_cluster = keos_cluster["spec"]["infra_provider"]
    if provider != provider_from_cluster:
        print(f"[WARN] Provider mismatch: detected '{provider}' from secrets but cluster reports '{provider_from_cluster}'")
        provider = provider_from_cluster

    print("[INFO] Provider: " + provider)

    if not config["dry_run"] and not config["yes"]:
        request_confirmation()

    # Check supported upgrades (provider already retrieved above)
    managed = keos_cluster["spec"]["control_plane"]["managed"]
    if not ((provider == "aws" and managed) or (provider == "azure" and not managed) or (provider == "gcp" and managed)):
        print("[ERROR] Upgrade is only supported for EKS, GKE and Azure VMs clusters")
        sys.exit(1)

    # Setting clusterctl env vars
    env_vars = "CLUSTER_TOPOLOGY=true CLUSTERCTL_DISABLE_VERSIONCHECK=true GOPROXY=off"

    # Get and update the helm repository if needed
    helm_repository_current = get_helm_repository(keos_cluster)
    helm_repository = input(f"The current helm repository is: {helm_repository_current}. Do you want to indicate a new helm repository? Press enter or specify new repository: ")
    if helm_repository == "" or helm_repository == helm_repository_current:
        print("[INFO] Helm repository unchanged: SKIP")
    else:
        validate_helm_repository(helm_repository)
        update_helm_repository(cluster_name, helm_repository, config["dry_run"])

    # Scale down cluster-autoscaler to avoid issues during the upgrade process
    scale_cluster_autoscaler(0, config["dry_run"])

    # Configure provider-specific environment variables and credentials for clusterctl
    print("[INFO] Configuring provider-specific environment variables for clusterctl:", end=" ", flush=True)

    if provider == "aws":
        # AWS/CAPA (Cluster API Provider AWS) configuration
        namespace = "capa-system"
        version = CAPA
        # Extract AWS credentials from Kubernetes secret
        credentials = subprocess.getoutput(kubectl + " -n " + namespace + " get secret capa-manager-bootstrap-credentials -o jsonpath='{.data.credentials}'")
        # Enable EKS IAM integration and set base64-encoded credentials
        env_vars += " CAPA_EKS_IAM=true AWS_B64ENCODED_CREDENTIALS=" + credentials

    elif provider == "gcp":
        # GCP/CAPG (Cluster API Provider GCP) configuration
        namespace = "capg-system"
        version = CAPG
        # Extract GCP service account credentials from Kubernetes secret
        credentials = subprocess.getoutput(kubectl + " -n " + namespace + " get secret capg-manager-bootstrap-credentials -o json | jq -r '.data[\"credentials.json\"]'")
        # Enable experimental features for managed GKE clusters
        if managed:
            env_vars += " EXP_MACHINE_POOL=true EXP_CAPG_GKE=true"
        # Set base64-encoded GCP credentials
        env_vars += " GCP_B64ENCODED_CREDENTIALS=" + credentials

    elif provider == "azure":
        # Azure/CAPZ (Cluster API Provider Azure) configuration
        namespace = "capz-system"
        version = CAPZ
        # Enable experimental machine pool support for managed clusters
        if managed:
            env_vars += " EXP_MACHINE_POOL=true"

        # Configure Azure service principal credentials from vault secrets
        if "credentials" in vault_secrets_data["secrets"]["azure"]:
            credentials = vault_secrets_data["secrets"]["azure"]["credentials"]
            # Encode Azure credentials in base64 format for clusterctl
            env_vars += " AZURE_CLIENT_ID_B64=" + base64.b64encode(credentials["client_id"].encode("ascii")).decode("ascii")
            env_vars += " AZURE_CLIENT_SECRET_B64=" + base64.b64encode(credentials["client_secret"].encode("ascii")).decode("ascii")
            env_vars += " AZURE_SUBSCRIPTION_ID_B64=" + base64.b64encode(credentials["subscription_id"].encode("ascii")).decode("ascii")
            env_vars += " AZURE_TENANT_ID_B64=" + base64.b64encode(credentials["tenant_id"].encode("ascii")).decode("ascii")
        else:
            print("[ERROR] Azure credentials not found in secrets file")
            sys.exit(1)

    print("OK")

    # Set GITHUB_TOKEN env var if exists in vault secrets to avoid hitting github API rate limits during the upgrade process
    print("[INFO] Setting GITHUB_TOKEN environment:", end=" ", flush=True)
    if "github_token" in vault_secrets_data["secrets"]:
        env_vars += " GITHUB_TOKEN=" + vault_secrets_data["secrets"]["github_token"]
        helm = "GITHUB_TOKEN=" + vault_secrets_data["secrets"]["github_token"] + " " + helm
        kubectl = "GITHUB_TOKEN=" + vault_secrets_data["secrets"]["github_token"] + " " + kubectl
        print("OK")
    else:
        print("SKIP (not configured)")

    # Configure backup if not disabled
    if not config["disable_backup"]:
        now = datetime.now()
        backup_dir = backup_dir + now.strftime("%Y%m%d-%H%M%S")
        backup(backup_dir, namespace, cluster_name, config["dry_run"])
    else:
        print("[INFO] Backup disabled: SKIP")

    # Prepare capsule
    if not config["disable_prepare_capsule"]:
        prepare_capsule(config["dry_run"])
    else:
        print("[INFO] Capsule preparation disabled: SKIP")

    # Re-fetch KeosCluster and ClusterConfig to ensure we work with the latest state before upgrading
    print("[INFO] Re-fetching KeosCluster and ClusterConfig:", end=" ", flush=True)
    keos_cluster, cluster_config = get_keos_cluster_cluster_config()
    print("OK")

    private_registry = is_private_registry_enabled(cluster_config)
    private_helm_repo = is_private_helm_repo_enabled(cluster_config)
    cluster_operator_version = config["cluster_operator"]

    charts_to_upgrade = dict(common_charts)
    if provider == "aws":
        # Since aws-load-balancer-controller is optional we need to check if is installed
        aws_eks_charts_installed = filter_installed_charts(aws_eks_charts)
        charts_to_upgrade.update(aws_eks_charts_installed)
    elif provider == "azure":
        charts_to_upgrade.update(azure_vm_charts)
    charts_to_upgrade["cluster-operator"]["version"] = cluster_operator_version

    # Filter out charts that are not installed to avoid errors
    charts_to_upgrade = filter_installed_charts(charts_to_upgrade)

    upgrade_charts(charts_to_upgrade)
    print("[INFO] All charts updated successfully")

    # Restore capsule
    if not config["disable_prepare_capsule"]:
        restore_capsule(config["dry_run"])

    print("[INFO] Waiting for the cluster-operator helmrelease to be ready:", end=" ", flush=True)
    command = f"{kubectl} wait helmrelease cluster-operator -n kube-system --for=jsonpath='{{.status.conditions[?(@.type==\"Ready\")].status}}'=True --timeout=5m"
    try:
        run_command(command)
        print("OK")
    except Exception as e:
        print("[WARN] HelmRelease not ready, checking status...")
        status_cmd = f"{kubectl} get helmrelease cluster-operator -n kube-system -o jsonpath='{{.status}}'"
        status_output, _ = run_command(status_cmd, allow_errors=True)
        print(f"[INFO] HelmRelease status: {status_output}")
        raise e
    print("[INFO] Upgrading Cluster Operator components...")
    print("[INFO] Suspending cluster-operator helmrelease:", end =" ", flush=True)

    command = kubectl + " patch helmrelease cluster-operator -n kube-system --type merge --patch '{\"spec\":{\"suspend\":true}}'"
    run_command(command)
    print("OK")

    stop_keoscluster_controller()
    disable_keoscluster_webhooks()
    update_clusterconfig(cluster_config, charts_to_upgrade, provider, cluster_operator_version)

    # -------------------------------------------------
    # Private registry configuration for GCP (critical: must run before clusterctl upgrade)
    # -------------------------------------------------
    if private_registry:
        registry_url = get_keos_registry_url(keos_cluster)
        print(f"[DEBUG] Using private registry: {registry_url}")
        create_clusterctl_config_for_private_registry(registry_url, provider, pull_through=is_ecr_pull_through_enabled(keos_cluster))
        if provider == "gcp":
            # Also patch GCP CRDs to remove conversion webhooks (caBundle issue)
            config_dir = os.path.expanduser("~/.cluster-api")
            patch_gcp_crd_conversion_webhook(config_dir)

    # -------------------------------------------------
    # GCP CRD conversion webhook cleanup (prevents caBundle PEM error during clusterctl upgrade)
    # -------------------------------------------------
    if provider == "gcp":
        patch_capg_crds_live()
    # -------------------------------------------------
    # Execute clusterctl upgrade
    # -------------------------------------------------
    upgrade_cluster_api_providers(provider)
    print("[INFO] Cluster API providers upgraded successfully")

    print("[INFO] Restoring KEOSCluster webhooks and starting controller...")
    keos_cluster, cluster_config = get_keos_cluster_cluster_config()
    provider = keos_cluster["spec"]["infra_provider"]
    restore_keoscluster_webhooks()
    start_keoscluster_controller()
    print("[INFO] Resuming cluster-operator helmrelease:", end =" ", flush=True)
    command = kubectl + " patch helmrelease cluster-operator -n kube-system --type merge --patch '{\"spec\":{\"suspend\":false}}'"
    run_command(command)
    print("OK")

    print("[INFO] Waiting for the cluster-operator helmrelease to be ready:", end =" ", flush=True)
    command = kubectl + " wait helmrelease cluster-operator -n kube-system --for=condition=Ready --timeout=5m"
    try:
        run_command(command)
        print("OK")
    except Exception as e:
        print("FAILED")
        print("[ERROR] HelmRelease failed to become ready, checking status...")
        status_cmd = f"{kubectl} get helmrelease cluster-operator -n kube-system -o yaml"
        status_output, _ = run_command(status_cmd, allow_errors=True)
        print(f"[DEBUG] HelmRelease details:\n{status_output}")
        print("[HINT] Check if the Helm chart exists in the registry and credentials are correct")
        raise e

    cluster_name = keos_cluster["metadata"]["name"]

    print("[INFO] Waiting for keoscluster to be ready:", end =" ", flush=True)

    command = (
        kubectl + " wait --for=jsonpath=\"{.status.ready}\"=true KeosCluster "
        + cluster_name + " -n cluster-" + cluster_name + " --timeout 5m"
    )
    execute_command(command, False)

    command = kubectl + " wait deployment -n kube-system keoscluster-controller-manager --for=condition=Available --timeout=5m"
    try:
        run_command(command)
        print("[INFO] keoscluster-controller-manager is Available")
    except Exception as e:
        print("[ERROR] Failed to wait for keoscluster-controller-manager:", e)

    print("[INFO] Restoring cluster-autoscaler replicas")
    scale_cluster_autoscaler(2, config["dry_run"])

    end_time = time.time()
    elapsed_time = end_time - start_time
    minutes, seconds = divmod(elapsed_time, 60)
    print("[INFO] Upgrade process finished successfully in " + str(int(minutes)) + " minutes and " + "{:.2f}".format(seconds) + " seconds")
