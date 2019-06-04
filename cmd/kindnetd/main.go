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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// kindnetd is a simple networking daemon to complete kind's CNI implementation
// kindnetd will ensure routes to the other node's PodCIDR via their InternalIP
// kindnetd will also write a templated cni config supplied with PodCIDR
//
// input envs:
// - HOST_IP: hould be populated by downward API
// - POD_IP: should be populated by downward API
// - CNI_CONFIG_TEMPLATE: the cni .conflist template, run with {{ .PodCIDR }}

// TODO: improve logging & error handling

func main() {
	// create a client
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// obtain the host and pod ip addresses
	// if both ips are different we are not using the host network
	hostIP, podIP := os.Getenv("HOST_IP"), os.Getenv("POD_IP")
	fmt.Printf("hostIP = %s\npodIP = %s\n", hostIP, podIP)
	if hostIP != podIP {
		panic(fmt.Sprintf(
			"hostIP(= %q) != podIP(= %q) but must be running with host network: ",
			hostIP, podIP,
		))
	}

	// used to track if the cni config inputs changed and write the config
	cniConfigWriter := &CNIConfigWriter{
		path:     CNIConfigPath,
		template: os.Getenv("CNI_CONFIG_TEMPLATE"),
	}
	// setup nodes reconcile function, closes over arguments
	reconcileNodes := makeNodesReconciler(cniConfigWriter, hostIP)

	// main control loop
	for {
		// Gets the Nodes information from the API
		nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		// reconcile the nodes
		if err := reconcileNodes(nodes); err != nil {
			panic(err.Error())
		}

		// rate limit
		time.Sleep(10 * time.Second)
	}
}

// nodeNodesReconciler returns a reconciliation func for nodes
func makeNodesReconciler(cniConfigWriter *CNIConfigWriter, hostIP string) func(*corev1.NodeList) error {
	// reconciles a node
	reconcileNode := func(node corev1.Node) error {
		// first get this node's IP
		nodeIP := internalIP(node)
		if nodeIP == "" {
			fmt.Printf("Node %v has no Internal IP, ignoring\n", node.Name)
			return nil
		}

		// don't do anything unless there is a PodCIDR
		podCIDR := node.Spec.PodCIDR
		if podCIDR == "" {
			fmt.Printf("Node %v has no CIDR, ignoring\n", node.Name)
			return nil
		}

		// This is our node. We don't need to add routes, but we might need to
		// update the cni config.
		if nodeIP == hostIP {
			fmt.Printf("handling current node\n")
			// compute the current cni config inputs
			if err := cniConfigWriter.Write(
				ComputeCNIConfigInputs(node),
			); err != nil {
				return err
			}
			// we're done handling this node
			return nil
		}

		fmt.Printf("Handling node with IP: %s\n", nodeIP)
		// parse subnet
		dst, err := netlink.ParseIPNet(podCIDR)
		if err != nil {
			return err
		}
		fmt.Printf("Node %v has CIDR %s \n", node.Name, podCIDR)

		// Check if the route exists to the other node's PodCIDR
		ip := net.ParseIP(nodeIP)
		routeToDst := netlink.Route{Dst: dst, Gw: ip}
		route, err := netlink.RouteListFiltered(nl.GetIPFamily(ip), &routeToDst, netlink.RT_FILTER_DST)
		if err != nil {
			return err
		}

		// Add route if not present
		if len(route) == 0 {
			if err := netlink.RouteAdd(&routeToDst); err != nil {
				return err
			}
			fmt.Printf("Adding route %v \n", routeToDst)
		}

		return nil
	}

	// return a reconciler for all the nodes
	return func(nodes *corev1.NodeList) error {
		for _, node := range nodes.Items {
			if err := reconcileNode(node); err != nil {
				return err
			}
		}
		return nil
	}
}

// internalIP returns the internalIP address for node
func internalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			return address.Address
		}
	}
	return ""
}

/* cni config management */

// CNIConfigInputs is supplied to the CNI config template
type CNIConfigInputs struct {
	PodCIDR string
}

// ComputeCNIConfigInputs computes the template inputs for CNIConfigWriter
func ComputeCNIConfigInputs(node corev1.Node) CNIConfigInputs {
	podCIDR := node.Spec.PodCIDR
	return CNIConfigInputs{
		PodCIDR: podCIDR,
	}
}

// CNIConfigPath is where kindnetd will write the computed CNI config
const CNIConfigPath = "/etc/cni/net.d/10-kindnet.conflist"

// CNIConfigWriter no-ops re-writing config with the same inputs
// NOTE: should only be called from a single goroutine
type CNIConfigWriter struct {
	path       string
	template   string
	lastInputs CNIConfigInputs
}

// Write will write the config based on
func (c *CNIConfigWriter) Write(inputs CNIConfigInputs) error {
	if inputs == c.lastInputs {
		return nil
	}

	// use an extension not recognized by CNI to write the contents initially
	// https://github.com/containerd/go-cni/blob/891c2a41e18144b2d7921f971d6c9789a68046b2/opts.go#L170
	// then we can rename to atomically make the file appear
	f, err := os.Create(c.path + ".temp")
	if err != nil {
		return err
	}

	// actually write the config
	if err := writeCNIConfig(f, c.template, inputs); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Sync()
	f.Close()

	// then we can rename to the target config path
	if err := os.Rename(f.Name(), c.path); err != nil {
		return err
	}

	// we're safely done now, record the inputs
	c.lastInputs = inputs
	return nil
}

func writeCNIConfig(w io.Writer, rawTemplate string, data CNIConfigInputs) error {
	t, err := template.New("cni-json").Parse(rawTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to parse cni template")
	}
	return t.Execute(w, &data)
}
