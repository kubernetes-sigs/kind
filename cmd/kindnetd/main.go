/*
Copyright 2016 The Kubernetes Authors.

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
	"net"
	"os"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// obtain the host and pod ip addresses
	// if both ips are different we are not using the host network
	hostIP, podIP := os.Getenv("HOST_IP"), os.Getenv("POD_IP")
	if hostIP != podIP {
		panic(err.Error())
	}
	// initates the control loop
	for {
		// Gets the Nodes information from the API
		nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))

		// Iterate over all the nodes information
		for _, node := range nodes.Items {
			var nodeIP string

			// Obtain node internal IP
			for _, address := range node.Status.Addresses {
				if address.Type == "InternalIP" {
					nodeIP = address.Address
				}
			}

			// We don't need to install routes to our local subnet
			if nodeIP != hostIP {
				ip := net.ParseIP(nodeIP)
				// Obtain Pod Subnet
				if node.Spec.PodCIDR == "" {
					fmt.Printf("Node %v has no CIDR, ignoring\n", node.Name)
					continue
				}
				dst, err := netlink.ParseIPNet(node.Spec.PodCIDR)
				if err != nil {
					panic(err.Error())
				}
				fmt.Printf("Node %v has CIDR %s \n",
					node.Name, node.Spec.PodCIDR)

				// Check if the route exists
				routeToDst := netlink.Route{Dst: dst, Gw: ip}
				route, err := netlink.RouteListFiltered(nl.GetIPFamily(ip), &routeToDst, netlink.RT_FILTER_DST)
				if err != nil {
					panic(err.Error())
				}
				// Add route if not present
				if len(route) == 0 {
					if err := netlink.RouteAdd(&routeToDst); err != nil {
						panic(err.Error())
					}
					fmt.Printf("Adding route to the system %v \n", routeToDst)
				}
			}

		}

		// Writes the routes to the Pod Subnets in other nodes

		// Sleep
		time.Sleep(10 * time.Second)
	}
}
