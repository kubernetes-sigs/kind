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

// Package kubeadminit implements the kubeadm init action
package kubeadminit

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"

	"gopkg.in/yaml.v2"
)

// kubeadmInitAction implements action for executing the kubadm init
// and a set of default post init operations like e.g. install the
// CNI network plugin.
type action struct{}

// NewAction returns a new action for kubeadm init
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Starting control-plane 🕹️")
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

	// run kubeadm
	cmd := node.Command(
		// init because this is the control plane node
		"kubeadm", "init",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		// specify our generated config file
		"--config=/kind/kubeadm.conf",
		"--skip-token-print",
		// increase verbosity for debugging
		"--v=6",
	)
	lines, err := exec.CombinedOutputLines(cmd)
	log.Debug(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to init node with kubeadm")
	}

	// copies the kubeconfig files locally in order to make the cluster
	// usable with kubectl.
	// the kubeconfig file created by kubeadm internally to the node
	// must be modified in order to use the random host port reserved
	// for the API server and exposed by the node

	hostPort, err := getAPIServerPort(allNodes)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	kubeConfigPath := ctx.ClusterContext.KubeConfigPath()
	clusterName := ctx.ClusterContext.Name()
	if err := writeKubeConfig(node, kubeConfigPath, hostPort, clusterName); err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// if we are only provisioning one node, remove the master taint
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#master-isolation
	if len(allNodes) == 1 {
		if err := node.Command(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
			"taint", "nodes", "--all", "node-role.kubernetes.io/master-",
		).Run(); err != nil {
			return errors.Wrap(err, "failed to remove master taint")
		}
	}

	// mark success
	ctx.Status.End(true)
	return nil
}


type genericYaml map[string]interface{}

// writeKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
// While copying to the host machine the control plane address
// is replaced with local host and the control plane port with
// a randomly generated port reserved during node creation.
// We also make the user reference id unique to allow to use
// multiple kubeconfig files in the KUBECONFIG variable
func writeKubeConfig(n *nodes.Node, dest string, hostPort int32, userSuffix string) error {
	cmd := n.Command("cat", "/etc/kubernetes/admin.conf")
	buff, err := exec.Output(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	var config genericYaml
	if err = yaml.Unmarshal(buff, &config); err != nil {
		return errors.Wrap(err, "failed to parse kubeconfig file")
	}

	// Swap out the server for the forwarded localhost:port
	cluster := config["clusters"].([]interface{})[0].(map[interface{}]interface{})
	for k, v := range cluster {
		if k == "cluster" {
			clusterMap := v.(map[interface{}]interface{})
			for k, v = range clusterMap {
				if k == "server" {
					clusterMap[k] = fmt.Sprintf("https://localhost:%d", hostPort)
					break
				}
			}
			break
		}
	}

	// Must make the user reference id unique to our cluster to allow
	// for using the kubeconfig file of multiple clusters at the same
	// time in the KUBECONFIG variable.

	// Add a suffix to the user reference id in the context section
	// which is in the "contexts[0].context.user" field.
	context := config["contexts"].([]interface{})[0].(map[interface{}]interface{})
	var newUserName, oldUserName string
	for k, v := range context {
		if k == "context" {
			contextMap := v.(map[interface{}]interface{})
			for k, v = range contextMap {
				if k == "user" {
					oldUserName = fmt.Sprintf("%s", v)
					newUserName = fmt.Sprintf("%s-%s", oldUserName, userSuffix)
					contextMap[k] = newUserName
					break
				}
			}
			break
		}
	}

	// Use the new user reference id in the users section
	// which is in the "users[0].name" field.
	// In the same loop, add the 'username' field in the "users[0].user" section.
	user := config["users"].([]interface{})[0].(map[interface{}]interface{})
	for k := range user {
		if k == "name" {
			user[k] = newUserName
		} else if k == "user" {
			userMap := user[k].(map[interface{}]interface{})
			userMap["username"] = oldUserName
		}
	}
	buff, err = yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to create new yaml output for kubeconfig")
	}

	// create the directory to contain the KUBECONFIG file.
	// 0755 is taken from client-go's config handling logic: https://github.com/kubernetes/client-go/blob/5d107d4ebc00ee0ea606ad7e39fd6ce4b0d9bf9e/tools/clientcmd/loader.go#L412
	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create kubeconfig output directory")
	}

	return ioutil.WriteFile(dest, buff, 0600)
}

// getAPIServerPort returns the port on the host on which the APIServer
// is exposed
func getAPIServerPort(allNodes []nodes.Node) (int32, error) {
	// select the external loadbalancer first
	node, err := nodes.ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return 0, err
	}
	// node will be nil if there is no load balancer
	if node != nil {
		return node.Ports(loadbalancer.ControlPlanePort)
	}

	// fallback to the bootstrap control plane
	node, err = nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return 0, err
	}

	return node.Ports(kubeadm.APIServerPort)
}
