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

	"sigs.k8s.io/kind/pkg/fs"
)

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

	// used to track if the cni config inputs changed
	lastCNIData := cniConfigInputs{}

	// main control loop
	for {
		// Gets the Nodes information from the API
		nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))

		// Iterate over all the nodes information
		for _, node := range nodes.Items {
			// Obtain node internal IP
			nodeIP := internalIP(node)
			fmt.Printf("Handling node with IP: %s\n", nodeIP)

			// this is our node, handle it specially
			// we don't need to add routes, but we might need to update
			// the cni config
			if nodeIP == hostIP {
				fmt.Printf("handling current node\n")
				// compute the current cni config inputs
				cniData := computeCNIConfigInputs(node)
				// if it changed, write out the new config
				if cniData != lastCNIData {
					fmt.Printf("Writing CNI data\n")
					if err := writeCNIConfig(cniData); err != nil {
						panic(err.Error())
					}
					lastCNIData = cniData
				}
				// we're don handling this node
				continue
			}

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
			ip := net.ParseIP(nodeIP)
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

		// rate limit
		time.Sleep(10 * time.Second)
	}
}

func internalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			return address.Address
		}
	}
	return ""
}

func computeCNIConfigInputs(node corev1.Node) cniConfigInputs {
	podCIDR := node.Spec.PodCIDR
	return cniConfigInputs{
		PodCIDR: podCIDR,
	}
}

func writeCNIConfig(data cniConfigInputs) error {
	return fs.AtomicWriteFile("/etc/cni/net.d/10-kindnet.conflist", func(w io.Writer) error {
		t, err := template.New("cni-json").Parse(cniTemplate)
		if err != nil {
			panic(errors.Wrap(err, "failed to parse cni template"))
		}
		return t.Execute(w, &data)
	})
}

type cniConfigInputs struct {
	PodCIDR string
}

const cniTemplate = `
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
        ],
        "ranges": [
          [
            {
               "subnet": "{{ .PodCIDR }}"
            }
          ]
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
`
