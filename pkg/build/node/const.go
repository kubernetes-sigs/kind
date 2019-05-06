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

package node

// these are well known paths within the node image
const (
	// TODO: refactor kubernetesVersionLocation to a common internal package
	kubernetesVersionLocation  = "/kind/version"
	defaultCNIManifestLocation = "/kind/manifests/default-cni.yaml"
)

/*
The default CNI manifest and images are created by the kind community currently.

Kind leverages the standard networking plugins https://github.com/containernetworking/plugins
and install a daemon in each node that creates the corresponding routes to each others pod
subnets

Basically, the behavior is as follows:
- kubernetes does the podSubnet assignement to each node
- the container plugin is done using the standard "bridge" plugin
- it sends the subnet information to IPAM "host-local" plugin using "ipRanges" capability
- the bridge plugin has enable the harpain feature to allow the pods to reach the services VIPs
- the kindnet daemon watches the nodes and the podSubnets assigned and install the corresponding
routes in each node

*/

var defaultCNIImages = []string{"aojea/kindnet", "busybox"}

const defaultCNIManifest = `# https://raw.githubusercontent.com/aojea/kindnet/master/install-kindnet.yaml
---
apiVersion: extensions/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp.kindnet.unprivileged
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: docker/default
    seccomp.security.alpha.kubernetes.io/defaultProfileName: docker/default
    apparmor.security.beta.kubernetes.io/allowedProfileNames: runtime/default
    apparmor.security.beta.kubernetes.io/defaultProfileName: runtime/default
spec:
  privileged: false
  volumes:
    - configMap
    - secret
    - emptyDir
    - hostPath
  allowedHostPaths:
    - pathPrefix: "/etc/cni/net.d"
  readOnlyRootFilesystem: false
  # Users and groups
  runAsUser:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  # Privilege Escalation
  allowPrivilegeEscalation: false
  defaultAllowPrivilegeEscalation: false
  # Capabilities
  allowedCapabilities: ['NET_ADMIN']
  defaultAddCapabilities: []
  requiredDropCapabilities: []
  # Host namespaces
  hostPID: false
  hostIPC: false
  hostNetwork: true
  hostPorts:
  - min: 0
    max: 65535
  # SELinux
  seLinux:
    rule: 'RunAsAny'
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kindnet
rules:
  - apiGroups: ['extensions']
    resources: ['podsecuritypolicies']
    verbs: ['use']
    resourceNames: ['psp.kindnet.unprivileged']
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kindnet
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kindnet
subjects:
- kind: ServiceAccount
  name: kindnet
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kindnet
  namespace: kube-system
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: kindnet-cfg
  namespace: kube-system
  labels:
    tier: node
    app: kindnet
data:
  cni-conf.json: |
    {
      "cniVersion": "0.3.1",
      "name": "kindOfBridge",
      "plugins": [
        {
          "type": "bridge",
          "bridge": "kind0",
          "capabilities": {"ipRanges": true},
          "isGateway": true,
          "hairpinMode": true,
          "ipMasq": false,
          "ipam": {
            "type": "host-local",
            "routes": [
              {"dst": "0.0.0.0/0"},
              {"dst": "::/0"}
            ]
          }
        },
        {
          "type": "portmap",
          "capabilities": {"portMappings": true},
          "snat": false
        }
      ]
    }
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: kindnet-ds
  namespace: kube-system
  labels:
    tier: node
    app: kindnet 
spec:
  template:
    metadata:
      labels:
        tier: node
        app: kindnet
    spec:
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: kindnet
      initContainers:
      - name: install-cni-cfg
        image: busybox
        command: ["cp"]
        args: ["-f", "/kind/kube-kindnet/cni-conf.json", "/etc/cni/net.d/10-kindnet.conflist"]
        volumeMounts:
        - name: cni-cfg
          mountPath: /etc/cni/net.d
        - name: kindnet-cfg
          mountPath: /kind/kube-kindnet/
      containers:
      - name: kindnet-cni
        image: aojea/kindnet
        env:
          - name: HOST_IP
            valueFrom:
              fieldRef:
                fieldPath: status.hostIP
          - name: POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
          limits:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: false
          capabilities:
             add: ["NET_ADMIN"]
      volumes:
        - name: cni-cfg
          hostPath:
            path: /etc/cni/net.d
        - name: kindnet-cfg
          configMap:
            name: kindnet-cfg
---
`
