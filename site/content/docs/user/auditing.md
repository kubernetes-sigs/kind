---
title: "Auditing"
menu:
  main:
    parent: "user"
    identifier: "user-auditing"
    weight: 4
description: |-
    This guide covers how to enable Kubernetes API [auditing] on a kind cluster.

    [auditing]: https://kubernetes.io/docs/tasks/debug-application-cluster/audit/
---

## Overview

Kubernetes auditing provides a security-relevant, chronological set of records documenting the sequence of actions in a cluster. Auditing requires a file to define the [audit policy] and a backend configuration to store the logged events. Auditing supports two types of backends: log (file) & webhook. The following exercise uses the log backend.

Steps:

- Create the local audit-policy file
- Mount the local audit-policy file into the kind control plane
- Expose the control plane mounts to the API server
- Enable the auditing API flags
- Create a cluster

## Setup

### Create an `audit-policy.yaml` file

The [audit policy] defines the level of granularity outputted by the Kubernetes API server. The example below logs all requests at the "Metadata" level. See the [audit policy] docs for more examples. 

{{< codeFromInline lang="bash" >}}
cat <<EOF > audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
EOF
{{< /codeFromInline >}}

### Create a `kind-config.yaml` file.

To enable audit logging, use kind's [configuration file] to pass additional setup instructions. Kind uses `kubeadm` to provision the cluster and the configuration file has the ability to pass `kubeadmConfigPatches` for further customization.

{{< codeFromInline lang="bash" >}}
cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
        # enable auditing flags on the API server
        extraArgs:
          audit-log-path: /var/log/kubernetes/kube-apiserver-audit.log
          audit-policy-file: /etc/kubernetes/policies/audit-policy.yaml
        # mount new files / directories on the control plane
        extraVolumes:
          - name: audit-policies
            hostPath: /etc/kubernetes/policies
            mountPath: /etc/kubernetes/policies
            readOnly: true
            pathType: "DirectoryOrCreate"
          - name: "audit-logs"
            hostPath: "/var/log/kubernetes"
            mountPath: "/var/log/kubernetes"
            readOnly: false
            pathType: DirectoryOrCreate
  # mount the local file on the control plane
  extraMounts:
  - hostPath: ./audit-policy.yaml
    containerPath: /etc/kubernetes/policies/audit-policy.yaml
    readOnly: true
EOF
{{< /codeFromInline >}}

## Launch a new cluster

{{< codeFromInline lang="bash" >}}
kind create cluster --config kind-config.yaml
{{< /codeFromInline >}}

## View audit logs

Once the cluster is running, view the log files on the control plane in `/var/log/kubernetes/kube-apiserver-audit.log`.

{{< codeFromInline lang="bash" >}}
docker exec kind-control-plane cat /var/log/kubernetes/kube-apiserver-audit.log
{{< /codeFromInline >}}

## Troubleshooting

If logs are not present, let's ensure a few things are in place.

### Is the local audit-policy file mounted in the control-plane?

{{< codeFromInline lang="bash" >}}
docker exec kind-control-plane ls /etc/kubernetes/policies
{{< /codeFromInline >}}

Expected output:

```bash
audit-policy.yaml
```

### Does the API server contain the mounts and arguments?

{{< codeFromInline lang="bash" >}}
docker exec kind-control-plane cat /etc/kubernetes/manifests/kube-apiserver.yaml | grep audit
{{< /codeFromInline >}}

Expected output:

```bash
    - --audit-log-path=/var/log/kubernetes/kube-apiserver-audit.log
    - --audit-policy-file=/etc/kubernetes/policies/audit-policy.yaml
      name: audit-logs
      name: audit-policies
    name: audit-logs
    name: audit-policies
```

If the control plane requires further debugging use `docker exec -it kind-control-plane bash` to start an interactive terminal session with the container.

[audit policy]: https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#audit-policy
[configuration file]: /docs/user/configuration
