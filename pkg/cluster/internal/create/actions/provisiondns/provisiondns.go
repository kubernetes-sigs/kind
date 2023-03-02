/*
Copyright 2022 Spectrocloud Authors.

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

// Package waitforready implements the wait for ready action
package provisiondns

import (
	"bytes"
	"fmt"
	"os"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sort"
	"strconv"
	"strings"

	"time"
)

// Action implements an action for waiting for the cluster to be ready
type Action struct {
	waitTime time.Duration
}

// NewAction returns a new action for waiting for the cluster to be ready
func NewAction(waitTime time.Duration) actions.Action {
	return &Action{
		waitTime: waitTime,
	}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("SpectroCloud preflight check 📡")
	defer ctx.Status.End(false)
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}
	// get a control plane node to use to check cluster status
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0]
	if err := a.recreateDnsResources(ctx, node); err != nil {
		return err
	}
	ctx.Status.End(true)
	return nil
}

func (a *Action) recreateDnsResources(ctx *actions.ActionContext, node nodes.Node) error {
	//deleting kube-root-ca.crt in kube-system namespace
	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "cm", "kube-root-ca.crt", "-n", "kube-system",
	).Run()

	//deleting kube-root-ca.crt in kube-public namespace
	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "cm", "kube-root-ca.crt", "-n", "kube-public",
	).Run()

	//If etcd data is persisted locally then nodes information will also be present
	//In case if kind cluster is created with diff name then node name will change and k8s will now have older node and new node
	//Since, for older node, kubelet won't be present(as node is deleted), thus few pods gets stuck in Terminating state and
	//in this case older node has to be terminated
	//So, I am trying to list all nodes and sort it based on age and deleting older nodes (more then what is defined in kind config)
	var getNodesWriter bytes.Buffer
	cmd := node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"get", "nodes",
	)
	cmd.SetStdout(&getNodesWriter)
	cmd.SetStderr(os.Stderr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get nodes. %v", err)
	}

	getNodesOutput := getNodesWriter.String()
	lines := strings.Split(getNodesOutput, "\n")
	nodes := make(map[string]string)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		line = strings.Join(strings.Fields(line), " ")
		values := strings.Split(line, " ")
		if values[0] == "NAME"{
			continue
		}
		nodes[ageToDuration(values[3]).String()] = values[0]
	}

	if len(nodes) > 1 {
		keys := make([]string, 0, len(nodes))
		for key := range nodes {
			keys = append(keys, key)
		}
		sort.SliceStable(keys, func(i, j int) bool{
			return ageToDuration(keys[i]) < ageToDuration(keys[j])
		})

		//Nodes more than currect total nodes will be removed from kind cluster
		allNodes, _ := ctx.Nodes()
		args := []string{"--kubeconfig=/etc/kubernetes/admin.conf", "delete", "nodes"}
		for i := len(allNodes); i < len(keys); i++ {
			args = append(args, nodes[keys[i]])
		}

		//ctx.Logger.V(1).Info(fmt.Sprintf("node names to be deleted: %v", args))
		delNodesCmd := node.Command("kubectl", args...)
		delNodesCmd.SetStderr(os.Stderr)
		if err := delNodesCmd.Run(); err != nil {
			ctx.Logger.Error(fmt.Sprintf("failed to delete older nodes %s", err.Error()))
			return err
		}
	}

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "pods", "-l", "k8s-app=kube-proxy", "-n", "kube-system",
	).Run()

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"rollout", "restart", "deployment/coredns", "-n", "kube-system",
	).Run()

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"wait", "--for=condition=available", "--timeout=300s", "deployment/coredns", "-n", "kube-system",
	).Run()

	return nil
}

func ageToDuration(d string) time.Duration {
	duration, _ := time.ParseDuration(k8sAgeToStdFmt(d))
	return duration
}

func k8sAgeToStdFmt(d string) string {
	if strings.Contains(d, "d"){
		hours := int64(0)
		dd := strings.Split(d, "d")
		noOfDays, _ := strconv.ParseInt(dd[0], 10, 32)
		hours = noOfDays * 24
		if len(dd) > 1 && len(dd[1]) > 0 {
			hh := strings.Split(dd[1], "h")
			noOfHours, _ := strconv.ParseInt(hh[0], 10, 32)
			hours = hours + noOfHours
		}
		return fmt.Sprintf("%dh", hours)
	} else if strings.Contains(d, "h") && strings.Contains(d, "m"){
		minutes := int64(0)
		hh := strings.Split(d, "h")
		noOfHours, _ := strconv.ParseInt(hh[0], 10, 32)
		minutes = noOfHours * 60
		if len(hh) > 1 && len(hh[1]) > 0 {
			mm := strings.Split(hh[1], "m")
			noOfMins, _ := strconv.ParseInt(mm[0], 10, 32)
			minutes = minutes + noOfMins
		}
		return fmt.Sprintf("%dm", minutes)
	} else if strings.Contains(d, "m") && strings.Contains(d, "s"){
		seconds := int64(0)
		mm := strings.Split(d, "m")
		noOfMins, _ := strconv.ParseInt(mm[0], 10, 32)
		seconds = noOfMins * 60
		if len(mm) > 1 && len(mm[1]) > 0 {
			ss := strings.Split(mm[1], "s")
			noOfSeconds, _ := strconv.ParseInt(ss[0], 10, 32)
			seconds = seconds + noOfSeconds
		}
		return fmt.Sprintf("%ds", seconds)
	}
	return d
}
