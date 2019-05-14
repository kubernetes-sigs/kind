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
The default CNI manifest and images are our own tiny kindnet
*/

var defaultCNIImages = []string{"kindest/kindnetd:0.1.0", "k8s.gcr.io/ip-masq-agent:v2.4.0"}

const defaultCNIManifest = `
# kindnetd networking manifest
# would you kindly template this file
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: kindnet
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
  - apiGroups:
    - policy
    resources:
    - podsecuritypolicies
    verbs:
    - use
    resourceNames: 
    - kindnet
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
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
    k8s-app: kindnet
data:
  cni-conf-template.json: |
    {
      "cniVersion": "0.3.1",
      "name": "kindnet",
      "plugins": [
        {
          "type": "bridge",
          "ipMasq": false,
          "isDefaultGateway": true,
          "hairpinMode": true,
          "ipam": {
            "type": "host-local",
            "dataDir": "/run/cni-ipam-state",
            "routes": [
              {"dst": "0.0.0.0/0"},
              {"dst": "::/0"}
            ],
            "ranges": [
              [
                {
                  "subnet": "{{"{{ .PodCIDR }}"}}"
                }
              ]
            ]
          }
        },
        {
          "type": "portmap",
          "capabilities": {
            "portMappings": true
          }
        }
      ]
    }
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: kindnet
  namespace: kube-system
  labels:
    tier: node
    app: kindnet
    k8s-app: kindnet
spec:
  template:
    metadata:
      labels:
        tier: node
        app: kindnet
        k8s-app: kindnet
    spec:
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: kindnet
      containers:
      - name: kindnet-cni
        image: kindest/kindnetd:0.1.0
        env:
          - name: HOST_IP
            valueFrom:
              fieldRef:
                fieldPath: status.hostIP
          - name: POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
          - name: CNI_CONFIG_TEMPLATE
            valueFrom:
              configMapKeyRef:
                name: kindnet-cfg
                key: cni-conf-template.json
        volumeMounts:
        - name: cni-cfg
          mountPath: /etc/cni/net.d
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
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: ip-masq-agent
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: docker/default
    seccomp.security.alpha.kubernetes.io/defaultProfileName: docker/default
    apparmor.security.beta.kubernetes.io/allowedProfileNames: runtime/default
    apparmor.security.beta.kubernetes.io/defaultProfileName: runtime/default
spec:
  privileged: false
  volumes:
    - configMap
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
  allowedCapabilities: ['NET_RAW', 'NET_ADMIN']
  defaultAddCapabilities: []
  requiredDropCapabilities: []
  # Host namespaces
  hostPID: false
  hostIPC: false
  hostNetwork: true
  # SELinux
  seLinux:
    rule: 'RunAsAny'
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: ip-masq-agent
rules:
  - apiGroups:
      - policy
    resources:
      - podsecuritypolicies
    verbs:
      - use
    resourceNames:
      - ip-masq-agent
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: ip-masq-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ip-masq-agent
subjects:
- kind: ServiceAccount
  name: ip-masq-agent
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ip-masq-agent
  namespace: kube-system
---
# ipmasq agent
kind: ConfigMap
apiVersion: v1
metadata:
  name: ip-masq-agent
  namespace: kube-system
  labels:
    tier: node
    app: ip-masq-agent
    k8s-app: ip-masq-agent
data:
  config: |-
    nonMasqueradeCIDRs:
      - {{ .PodSubnet }}
    masqLinkLocal: false
    resyncInterval: 60s
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: ip-masq-agent
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        tier: node
        app: ip-masq-agent
        k8s-app: ip-masq-agent
    spec:
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: ip-masq-agent
      containers:
      - name: ip-masq-agent
        image: k8s.gcr.io/ip-masq-agent:v2.4.0
        securityContext:
          privileged: false
          capabilities:
            add: ["NET_RAW", "NET_ADMIN"]
        volumeMounts:
          - name: config
            mountPath: /etc/config
      volumes:
        - name: config
          configMap:
            # Note this ConfigMap must be created in the same namespace as the daemon pods - this spec uses kube-system
            name: ip-masq-agent
            optional: true
            items:
              # The daemon looks for its config in a YAML file at /etc/config/ip-masq-agent
              - key: config
                path: ip-masq-agent
---
`
