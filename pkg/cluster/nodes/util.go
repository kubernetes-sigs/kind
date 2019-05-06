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

package nodes

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
)

// GetControlPlaneEndpoint returns the control plane endpoint in case the
// cluster has an external load balancer in front of the control-plane nodes,
// otherwise return an empty string.
func GetControlPlaneEndpoint(allNodes []Node) (string, error) {
	node, err := ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return "", err
	}
	// if there is no external load balancer return the empty string
	if node == nil {
		return "", nil
	}

	// get the IP and port for the load balancer
	loadBalancerIP, err := node.IP()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get IP for node: %s", node.Name())
	}

	return fmt.Sprintf("%s:%d", loadBalancerIP, loadbalancer.ControlPlanePort), nil
}

// to avoid the DNS crash we need copy host's /etc/resolv.conf to node
// ref: https://github.com/kubernetes-sigs/kind/pull/484#issuecomment-489469044
func addResolve(node *Node) error {
	resolv, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		return errors.Wrap(err, "failed to read /etc/resolv.conf")
	}

	// filter the loopback addresses
	re := regexp.MustCompile("(?m)[\r\n]+^.*((127.([0-9]{1,3}.){2}[0-9]{1,3})|(::1)).*$")
	content := re.ReplaceAllString(string(resolv), "")

	err = node.WriteFile("/kind/resolv.conf", content)
	if err != nil {
		return errors.Wrap(err, "failed to write /kind/resolv.conf to node")
	}

	return nil
}
