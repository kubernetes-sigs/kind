---
title: "Pod Security Policy"
menu:
  main:
    parent: "user"
    identifier: "pod-security-policy"
    weight: 3
---

# Pod Security Policy

This guide covers setting up [Pod Security Policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/)
on a kind cluster.

## Enabling Pod Security Policy

Pod security policy control is implemented as an optional (but recommended) admission controller. PodSecurityPolicies are enforced by enabling the admission controller, but doing so without authorizing any policies will prevent any pods from being created in the cluster.

### Create Cluster

Create a kind cluster with the `PodSecurityPolicy` admission plugin enabled.

{{< codeFromInline lang="bash" >}}
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterConfiguration
    metadata:
      name: config
    apiServer:
      extraArgs:
        enable-admission-plugins: PodSecurityPolicy
- role: worker
EOF
{{< /codeFromInline >}}

### Install default security policies

However, the cluster is not ready to be used at this stage, the nodes are not ready:

{{< codeFromInline lang="bash" >}}
kubectl get nodes -A
NAME                 STATUS     ROLES    AGE   VERSION
kind-control-plane   NotReady   master   94s   v1.17.2
kind-worker          NotReady   <none>   52s   v1.17.2
{{< /codeFromInline >}}

and there is no Pods:

{{< codeFromInline lang="bash" >}}
kubectl get pods -A
No resources found.
{{< /codeFromInline >}}

We need to grant the controller access to the policy so it can create the system pods, like CoreDNS, kube-scheduler, ...

Deploy [Authorization Policies](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#authorizing-policies).

{{< codeFromInline lang="bash" >}}
cat <<EOF | kubectl apply -f -
piVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp-privileged
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: '*'
spec:
  privileged: true
  allowPrivilegeEscalation: true
  allowedCapabilities:
  - '*'
  volumes:
  - '*'
  hostNetwork: true
  hostPorts:
  - min: 0
    max: 65535
  hostIPC: true
  hostPID: true
  runAsUser:
    rule: 'RunAsAny'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
---
# Cluster role which grants access to the default pod security policy
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: psp-privileged
rules:
- apiGroups:
  - policy
  resourceNames:
  - psp-privileged
  resources:
  - podsecuritypolicies
  verbs:
  - use
---
# Cluster role binding for default pod security policy granting all authenticated users access
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: psp-privileged
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: psp-privileged
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:authenticated
EOF
{{< /codeFromInline >}}


after a few seconds you can observe how the system pods are being created and your cluster is ready to be used with Pod Security Policy enabled.

{{< codeFromInline lang="bash" >}}
kubectl get pods -A
NAMESPACE            NAME                                         READY   STATUS    RESTARTS   AGE
kube-system          coredns-6955765f44-f5572                     0/1     Pending   0          25s
kube-system          coredns-6955765f44-tt2fl                     0/1     Pending   0          25s
kube-system          etcd-kind-control-plane                      1/1     Running   0          26s
kube-system          kube-apiserver-kind-control-plane            1/1     Running   0          17s
kube-system          kube-controller-manager-kind-control-plane   1/1     Running   0          35s
kube-system          kube-scheduler-kind-control-plane            1/1     Running   0          38s
local-path-storage   local-path-provisioner-85445b74d4-592wd      0/1     Pending   0          25s

{{< /codeFromInline >}}