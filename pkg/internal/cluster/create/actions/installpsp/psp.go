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

// Package installpsp implements the an action to isntall a default
// Pod Security Policy
package installpsp

import (
	"strings"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/internal/cluster/create/actions"
)

type action struct{}

// NewAction returns a new action for installing storage
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Installing Default PodSecurityPolicy  💾")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	node, err := nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return err
	}

	// add the default storage class
	if err := addDefaultPodSecurityPolicy(node); err != nil {
		return errors.Wrap(err, "failed to add default Pod Security Policy")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// a default pod security policy unrestricted
// we need this for e2es (PodSecurityPolicy)
const defaultPodSecurityPolicyManifest = `# least restricted policy
---
apiVersion: policy/v1beta1
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
`

func addDefaultPodSecurityPolicy(controlPlane *nodes.Node) error {
	in := strings.NewReader(defaultPodSecurityPolicyManifest)
	cmd := controlPlane.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf", "apply", "-f", "-",
	)
	cmd.SetStdin(in)
	return cmd.Run()
}
