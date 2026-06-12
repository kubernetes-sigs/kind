#!/usr/bin/env python3
# -*- coding: utf-8 -*-

##############################################################
# Author: Stratio Clouds <clouds-integration@stratio.com>    #
# Supported provisioner versions: 0.17.0-0.8.X              #
# Supported cloud providers:                                 #
#   - EKS (AWS managed)                                      #
##############################################################

__version__ = "0.1.0"

import argparse
import json
import os
import subprocess
import sys
import time

from ansible_vault import Vault

CAPA_NAMESPACE = "capa-system"
CAPA_DEPLOYMENT = "capa-controller-manager"
CAPA_CONTAINER_INDEX = 0
KEOS_CLUSTER_NAMESPACE_PREFIX = "cluster-"

# Target and minimum versions for this migration
TARGET_CLUSTER_OPERATOR_VERSION = "0.7.0-PR317-SNAPSHOT"
MIN_CLUSTER_OPERATOR_VERSION = "0.6.1"
MIN_CAPA_VERSION = "v2.9.2"

# Feature gates required for MachinePool support
REQUIRED_FEATURE_GATES = {
    "MachinePool": "true",
    "EKSAllowAddRoles": "true",
}

kubectl = "kubectl"


# ---------------------------------------------------------------------------
# Utilities
# ---------------------------------------------------------------------------

def run_command(command, allow_errors=False):
    '''Run a shell command, return (output, returncode).'''
    status, output = subprocess.getstatusoutput(command)
    if status != 0 and not allow_errors:
        print("FAILED")
        print(f"[ERROR] {output}")
        sys.exit(1)
    return output, status


def _version_gte(version, minimum):
    '''Return True if version >= minimum, comparing semver-style (strips leading "v").'''
    def parse(v):
        return [int(x) for x in v.lstrip("v").split(".")[:3] if x.isdigit()]
    try:
        return parse(version) >= parse(minimum)
    except Exception:
        return False


def execute_command(command, dry_run, print_result=True):
    '''Execute a command respecting dry-run mode.'''
    if dry_run:
        if print_result:
            print("DRY-RUN")
        return ""
    output, _ = run_command(command)
    if print_result:
        print("OK")
    return output


# ---------------------------------------------------------------------------
# S4 — Prerequisite validation
# ---------------------------------------------------------------------------

def validate_prerequisites(dry_run):
    '''Validate that the cluster is ready for MachinePool migration.'''

    print("[INFO] Validating prerequisites...")

    # 1. Detect provider and managed mode
    print("[INFO] Checking cluster provider:", end=" ", flush=True)
    cmd = kubectl + " get keoscluster -A -o jsonpath='{.items[0].spec.infra_provider}'"
    provider, _ = run_command(cmd, allow_errors=True)
    provider = provider.strip().strip("'")
    if provider != "aws":
        print(f"FAILED\n[ERROR] Provider '{provider}' is not supported. Only 'aws' is supported.")
        sys.exit(1)
    print(f"OK ({provider})")

    print("[INFO] Checking managed control plane:", end=" ", flush=True)
    cmd = kubectl + " get keoscluster -A -o jsonpath='{.items[0].spec.control_plane.managed}'"
    managed, _ = run_command(cmd, allow_errors=True)
    managed = managed.strip().strip("'")
    if managed != "true":
        print("FAILED\n[ERROR] Only EKS managed clusters (control_plane.managed=true) are supported.")
        sys.exit(1)
    print("OK")

    # 2. Check cluster-operator minimum version
    print(f"[INFO] Checking cluster-operator version (>= {MIN_CLUSTER_OPERATOR_VERSION}):", end=" ", flush=True)
    cmd = kubectl + " get deployment keoscluster-controller-manager -n kube-system -o jsonpath='{.spec.template.spec.containers[0].image}'"
    co_image, _ = run_command(cmd, allow_errors=True)
    co_image = co_image.strip().strip("'")
    co_version = co_image.split(":")[-1] if ":" in co_image else ""
    if not co_version:
        print(f"FAILED\n[ERROR] Could not determine cluster-operator version from image '{co_image}'.")
        sys.exit(1)
    if not _version_gte(co_version, MIN_CLUSTER_OPERATOR_VERSION):
        print(f"FAILED\n[ERROR] cluster-operator version '{co_version}' is below minimum '{MIN_CLUSTER_OPERATOR_VERSION}'. "
              "Run upgrade-provisioner.py first.")
        sys.exit(1)
    print(f"OK ({co_version})")

    # 3. Check CAPA minimum version
    print(f"[INFO] Checking CAPA version (>= {MIN_CAPA_VERSION}):", end=" ", flush=True)
    cmd = kubectl + f" get deployment {CAPA_DEPLOYMENT} -n {CAPA_NAMESPACE} -o jsonpath='{{.spec.template.spec.containers[0].image}}'"
    capa_image, _ = run_command(cmd, allow_errors=True)
    capa_image = capa_image.strip().strip("'")
    capa_version = capa_image.split(":")[-1] if ":" in capa_image else ""
    # Normalize: strip any suffix after the semver (e.g. v2.9.2-keos.1 → v2.9.2)
    capa_semver = capa_version.split("-")[0] if "-" in capa_version else capa_version
    if not capa_semver:
        print(f"FAILED\n[ERROR] Could not determine CAPA version from image '{capa_image}'.")
        sys.exit(1)
    if not _version_gte(capa_semver, MIN_CAPA_VERSION):
        print(f"FAILED\n[ERROR] CAPA version '{capa_version}' is below minimum '{MIN_CAPA_VERSION}'. "
              "Run upgrade-provisioner.py first to upgrade CAPA.")
        sys.exit(1)
    print(f"OK ({capa_version})")

    # 4. Check KeosCluster status.ready
    print("[INFO] Checking KeosCluster status.ready:", end=" ", flush=True)
    cmd = kubectl + " get keoscluster -A -o jsonpath='{.items[0].status.ready}'"
    ready, _ = run_command(cmd, allow_errors=True)
    ready = ready.strip().strip("'")
    if ready != "true":
        print(f"FAILED\n[ERROR] KeosCluster status.ready={ready}. "
              "Resolve any pending reconciliation before migrating.")
        sys.exit(1)
    print("OK")

    print("[INFO] All prerequisites satisfied.")


# ---------------------------------------------------------------------------
# S2 — Patch CAPA feature gates
# ---------------------------------------------------------------------------

def _get_capa_args():
    '''Return the current args list of the CAPA manager container.'''
    cmd = (kubectl + f" get deployment {CAPA_DEPLOYMENT} -n {CAPA_NAMESPACE}"
           f" -o jsonpath='{{.spec.template.spec.containers[{CAPA_CONTAINER_INDEX}].args}}'")
    output, _ = run_command(cmd)
    return json.loads(output)


def _find_feature_gates_index(args):
    '''Return the index of the --feature-gates arg, or -1 if not found.'''
    for i, arg in enumerate(args):
        if arg.startswith("--feature-gates="):
            return i
    return -1


def _parse_feature_gates(arg):
    '''Parse "--feature-gates=K=V,K=V,..." into a dict.'''
    raw = arg.split("=", 1)[1]
    result = {}
    for pair in raw.split(","):
        k, v = pair.split("=", 1)
        result[k.strip()] = v.strip()
    return result


def _build_feature_gates_arg(gates):
    '''Reconstruct "--feature-gates=..." from a dict, preserving original key order.'''
    pairs = ",".join(f"{k}={v}" for k, v in gates.items())
    return f"--feature-gates={pairs}"


def patch_capa_feature_gates(dry_run):
    '''Enable MachinePool and EKSAllowAddRoles feature gates in CAPA, idempotently.'''

    print("[INFO] Checking CAPA feature gates:", end=" ", flush=True)

    args = _get_capa_args()
    idx = _find_feature_gates_index(args)

    if idx == -1:
        print("FAILED\n[ERROR] --feature-gates argument not found in CAPA deployment.")
        sys.exit(1)

    gates = _parse_feature_gates(args[idx])

    # Check if already correct — idempotent
    already_set = all(gates.get(k) == v for k, v in REQUIRED_FEATURE_GATES.items())
    if already_set:
        print("OK (already set, SKIP)")
        return

    # Apply required gates
    gates.update(REQUIRED_FEATURE_GATES)
    new_arg = _build_feature_gates_arg(gates)

    patch = json.dumps([{
        "op": "replace",
        "path": f"/spec/template/spec/containers/{CAPA_CONTAINER_INDEX}/args/{idx}",
        "value": new_arg
    }])

    print("")
    print(f"[INFO] Patching CAPA feature gates (MachinePool=true, EKSAllowAddRoles=true):", end=" ", flush=True)
    cmd = (kubectl + f" patch deployment {CAPA_DEPLOYMENT} -n {CAPA_NAMESPACE}"
           f" --type=json -p='{patch}'")
    execute_command(cmd, dry_run)

    if not dry_run:
        print("[INFO] Waiting for CAPA rollout:", end=" ", flush=True)
        cmd = (kubectl + f" rollout status deployment/{CAPA_DEPLOYMENT}"
               f" -n {CAPA_NAMESPACE} --timeout=3m")
        execute_command(cmd, dry_run)

        # Verify
        args_after = _get_capa_args()
        gates_after = _parse_feature_gates(args_after[_find_feature_gates_index(args_after)])
        for k, v in REQUIRED_FEATURE_GATES.items():
            if gates_after.get(k) != v:
                print(f"[ERROR] Verification failed: {k}={gates_after.get(k)} (expected {v})")
                sys.exit(1)
        print("[INFO] CAPA feature gates verified OK.")


# ---------------------------------------------------------------------------
# S3 — Update cluster-operator
# ---------------------------------------------------------------------------

def update_cluster_operator(cluster_operator_version, dry_run):
    '''Update cluster-operator HelmRelease and image tag ConfigMap, then wait.'''

    print(f"[INFO] Updating cluster-operator to {cluster_operator_version}...")

    # 1. Get current version from ConfigMap
    cmd = (kubectl + " get configmap 00-cluster-operator-helm-chart-default-values"
           " -n kube-system -o jsonpath='{.data.values\\.yaml}'")
    values_yaml, _ = run_command(cmd, allow_errors=True)

    current_tag = None
    for line in values_yaml.splitlines():
        if "tag:" in line:
            current_tag = line.strip().split("tag:")[-1].strip()
            break

    if current_tag == cluster_operator_version:
        print(f"[INFO] cluster-operator already at {cluster_operator_version}: SKIP")
        return

    print(f"[INFO] Updating image tag in ConfigMap ({current_tag} → {cluster_operator_version}):", end=" ", flush=True)
    cmd = (kubectl + " get configmap 00-cluster-operator-helm-chart-default-values"
           " -n kube-system -o json"
           f" | python3 -c \""
           "import json,sys; cm=json.load(sys.stdin);"
           f"cm['data']['values.yaml']=cm['data']['values.yaml'].replace('tag: {current_tag}','tag: {cluster_operator_version}');"
           "print(json.dumps(cm))\""
           " | " + kubectl + " apply -f -")
    execute_command(cmd, dry_run)

    # 2. Update HelmRelease chart version
    print(f"[INFO] Updating HelmRelease chart version:", end=" ", flush=True)
    cmd = (kubectl + " patch helmrelease cluster-operator -n kube-system"
           f" --type=merge -p '{{\"spec\":{{\"chart\":{{\"spec\":{{\"version\":\"{cluster_operator_version}\"}}}}}}}}'")
    execute_command(cmd, dry_run)

    # 3. Force reconciliation
    print("[INFO] Forcing HelmRelease reconciliation:", end=" ", flush=True)
    ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    cmd = (kubectl + " annotate helmrelease cluster-operator -n kube-system"
           f" reconcile.fluxcd.io/requestedAt={ts} --overwrite")
    execute_command(cmd, dry_run)

    if not dry_run:
        # 4. Wait for controller deployment
        print("[INFO] Waiting for cluster-operator deployment:", end=" ", flush=True)
        cmd = (kubectl + " wait deployment -n kube-system keoscluster-controller-manager"
               " --for=condition=Available --timeout=5m")
        execute_command(cmd, dry_run)

        # 5. Wait for KeosCluster ready
        print("[INFO] Waiting for KeosCluster ready:", end=" ", flush=True)
        cmd = (kubectl + " wait keoscluster --all -A"
               " --for=jsonpath='{.status.ready}'=true --timeout=5m")
        execute_command(cmd, dry_run)


# ---------------------------------------------------------------------------
# S5 — Check MP readiness and print drain commands
# ---------------------------------------------------------------------------

def check_mp_ready(worker_mp_name):
    '''
    Verify that a MachinePool worker has enough Ready nodes to absorb MD workloads,
    then print the drain commands for the equivalent MD worker.
    '''

    print(f"[INFO] Checking MachinePool worker '{worker_mp_name}'...")

    # Find MachinePool objects for this worker
    cmd = (kubectl + f" get machinepool -A -l cluster.x-k8s.io/deployment-name={worker_mp_name}"
           " -o jsonpath='{.items[*].status.readyReplicas}'")
    output, status = run_command(cmd, allow_errors=True)
    if status != 0 or not output.strip():
        # Try by name prefix
        cmd = (kubectl + f" get machinepool -A --no-headers"
               f" | grep '{worker_mp_name}'")
        output, status = run_command(cmd, allow_errors=True)
        if not output.strip():
            print(f"[ERROR] No MachinePool found for worker '{worker_mp_name}'. "
                  "Make sure you have added the MP worker to the KeosCluster descriptor and it has been reconciled.")
            sys.exit(1)

    # Get MP nodes
    cmd = (kubectl + f" get nodes -l cluster.x-k8s.io/deployment-name={worker_mp_name}"
           " --no-headers -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[-1].type,READY:.status.conditions[-1].status")
    nodes_output, _ = run_command(cmd, allow_errors=True)

    ready_nodes = []
    for line in nodes_output.strip().splitlines():
        parts = line.split()
        if len(parts) >= 3 and parts[1] == "Ready" and parts[2] == "True":
            ready_nodes.append(parts[0])

    if not ready_nodes:
        print(f"[ERROR] No Ready nodes found for MachinePool '{worker_mp_name}'. "
              "Wait for the MP nodes to be Ready before draining the MD.")
        sys.exit(1)

    print(f"[INFO] MachinePool '{worker_mp_name}' has {len(ready_nodes)} Ready node(s): {', '.join(ready_nodes)}")

    # Infer equivalent MD name (convention: worker_mp_name with -mp suffix → strip -mp)
    worker_md_name = worker_mp_name.replace("-mp", "-md") if "-mp" in worker_mp_name else worker_mp_name + "-md"

    # Get MD nodes
    cmd = (kubectl + f" get nodes -l cluster.x-k8s.io/deployment-name={worker_md_name}"
           " --no-headers -o custom-columns=NAME:.metadata.name")
    md_nodes_output, status = run_command(cmd, allow_errors=True)
    md_nodes = [l.strip() for l in md_nodes_output.strip().splitlines() if l.strip()] if status == 0 else []

    if not md_nodes:
        print(f"[WARN] No nodes found for MachineDeployment '{worker_md_name}'. "
              "Verify the MD worker name manually.")
    else:
        print(f"[INFO] MachineDeployment '{worker_md_name}' has {len(md_nodes)} node(s) to drain.")

    print("")
    print("=" * 70)
    print(f"  CAPACITY CHECK: {len(ready_nodes)} MP nodes Ready — proceed with drain? (verify manually)")
    print("=" * 70)
    print("")
    print("  Step 1 — Drain each MD node (run one at a time, verify pods after each):")
    print("")
    for node in md_nodes:
        print(f"    kubectl drain {node} --ignore-daemonsets --delete-emptydir-data --timeout=10m")
        print("")
    print("  Step 2 — Remove the MD worker from the KeosCluster descriptor:")
    print(f"    Delete worker entry with name='{worker_md_name}' from spec.worker_nodes")
    print(f"    Then: kubectl patch keoscluster <name> -n cluster-<name> --type=merge \\")
    print(f"          -p '{{\"spec\":{{\"worker_nodes\": [<updated list without {worker_md_name}]}}}}'")
    print("")
    print("  Step 3 — Verify no MD objects remain:")
    print(f"    kubectl get machinedeployment -A | grep {worker_md_name}")
    print("")


# ---------------------------------------------------------------------------
# Argument parsing and main
# ---------------------------------------------------------------------------

def configure_aws_credentials(vault_secrets_data):
    print("[INFO] Configuring AWS credentials:", end=" ", flush=True)
    aws_creds = vault_secrets_data['secrets']['aws']['credentials']
    os.environ["AWS_PAGER"] = ""
    os.environ["AWS_ACCESS_KEY_ID"] = aws_creds['access_key']
    os.environ["AWS_SECRET_ACCESS_KEY"] = aws_creds['secret_key']
    os.environ["AWS_DEFAULT_REGION"] = aws_creds['region']
    role_arn = aws_creds.get('role_arn')
    if role_arn:
        result = subprocess.run(
            ["aws", "sts", "assume-role", "--role-arn", role_arn, "--role-session-name", "migrate-session"],
            capture_output=True, text=True
        )
        if result.returncode != 0:
            print("FAILED")
            print(result.stderr)
            sys.exit(1)
        creds = json.loads(result.stdout)["Credentials"]
        os.environ["AWS_ACCESS_KEY_ID"] = creds["AccessKeyId"]
        os.environ["AWS_SECRET_ACCESS_KEY"] = creds["SecretAccessKey"]
        os.environ["AWS_SESSION_TOKEN"] = creds["SessionToken"]
    print("OK")


def load_secrets(secrets_path, vault_password):
    print("[INFO] Reading secrets file:", end=" ", flush=True)
    if not os.path.exists(secrets_path):
        print(f"FAILED\n[ERROR] Secrets file '{secrets_path}' not found.")
        sys.exit(1)
    try:
        vault = Vault(vault_password)
        data = vault.load(open(secrets_path).read())
        print("OK")
        return data
    except Exception as e:
        print(f"FAILED\n[ERROR] Could not decrypt secrets file: {e}")
        sys.exit(1)


def parse_args():
    parser = argparse.ArgumentParser(
        description="Migrate EKS worker nodes from MachineDeployments to MachinePools.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    parser.add_argument("-k", "--kubeconfig",
                        help="Path to kubeconfig file. Can also be set via $KUBECONFIG.",
                        default=None)
    parser.add_argument("-p", "--vault-password",
                        help="Vault password to decrypt secrets.yml.",
                        required=True)
    parser.add_argument("-s", "--secrets",
                        help="Path to the encrypted secrets file.",
                        default="secrets.yml")
    parser.add_argument("--cluster-operator-version",
                        help="Target cluster-operator version. Defaults to the version bundled with this release.",
                        default=TARGET_CLUSTER_OPERATOR_VERSION)
    parser.add_argument("--dry-run", action="store_true",
                        help="Print actions without executing them.")
    parser.add_argument("--check-ready", metavar="WORKER_MP_NAME",
                        help="Check that a MachinePool worker is Ready and print drain commands for its MD equivalent.")
    return parser.parse_args()


def main():
    global kubectl
    args = parse_args()

    if args.kubeconfig:
        kubectl = f"kubectl --kubeconfig {os.path.expanduser(args.kubeconfig)}"

    if args.dry_run:
        print("[INFO] Running in DRY-RUN mode — no changes will be applied.")

    vault_secrets_data = load_secrets(args.secrets, args.vault_password)
    configure_aws_credentials(vault_secrets_data)

    # Mode: --check-ready (migration assistant, can run standalone after preparation)
    if args.check_ready:
        check_mp_ready(args.check_ready)
        return

    # Mode: preparation (S4 → S2 → S3)
    print("=" * 70)
    print("  IMPORTANT: Before continuing, verify that the following artifacts")
    print(f"  are available in the registry/repository configured for this cluster:")
    print(f"    - cluster-operator image:      {args.cluster_operator_version}")
    print(f"    - cluster-operator Helm chart: {args.cluster_operator_version}")
    print("")
    print("  If the cluster uses a private registry or private Helm repository,")
    print("  ensure those artifacts have been pushed before running this script.")
    print("=" * 70)
    print("")
    answer = input("Have you verified the artifacts are available? [y/N] ").strip().lower()
    if answer != "y":
        print("[INFO] Aborted by user.")
        sys.exit(0)
    print("")
    validate_prerequisites(args.dry_run)
    patch_capa_feature_gates(args.dry_run)
    update_cluster_operator(args.cluster_operator_version, args.dry_run)

    print("")
    print("=" * 70)
    print("  PREPARATION COMPLETE")
    print("  CAPA feature gates enabled. cluster-operator updated.")
    print("")
    print("  Next steps (repeat for each worker you want to migrate):")
    print("  1. Add the new MP worker to the KeosCluster descriptor: omit node_image (ami_type is optional, defaults to BOTTLEROCKET_x86_64).")
    print("  2. Wait for the MP nodes to become Ready.")
    print("  3. Run: python3 migrate-workers-to-machinepool.py --check-ready <worker-mp-name>")
    print("  4. Execute the printed drain commands manually, one node at a time.")
    print("=" * 70)
    print("")
    print("RESULT: OK")


if __name__ == "__main__":
    main()
