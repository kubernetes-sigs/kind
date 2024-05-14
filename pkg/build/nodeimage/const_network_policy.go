/*
Copyright 2024 The Kubernetes Authors.

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

package nodeimage

/*
The default network policy manifest and images are https://github.com/kubernetes-sigs/kube-network-policies
*/

const networkPolicyImage = "registry.k8s.io/networking/kube-network-policies:v0.2.0"

var defaultNetworkPolicyImage = []string{networkPolicyImage}

const defaultNetworkPolicyManifest = `
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kube-network-policies
rules:
  - apiGroups:
      - ""
    resources:
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
     - "networking.k8s.io"
    resources:
      - networkpolicies
    verbs:
      - list
      - watch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kube-network-policies
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kube-network-policies
subjects:
- kind: ServiceAccount
  name: kube-network-policies
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-network-policies
  namespace: kube-system
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-network-policies
  namespace: kube-system
  labels:
    tier: node
    app: kube-network-policies
    k8s-app: kube-network-policies
spec:
  selector:
    matchLabels:
      app: kube-network-policies
  template:
    metadata:
      labels:
        tier: node
        app: kube-network-policies
        k8s-app: kube-network-policies
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirst
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: kube-network-policies
      containers:
      - name: kube-network-policies
        image: ` + networkPolicyImage + `
        args:
        - /bin/netpol
        - -v
        - "2"
        volumeMounts:
        - name: lib-modules
          mountPath: /lib/modules
          readOnly: true
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: true
          capabilities:
            add: ["NET_ADMIN"]
      volumes:
      - name: lib-modules
        hostPath:
          path: /lib/modules
---
`
