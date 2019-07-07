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

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	utilsnet "k8s.io/utils/net"
)

// kindnetd is a simple networking daemon to complete kind's CNI implementation
// kindnetd will ensure routes to the other node's PodCIDR via their InternalIP
// kindnetd will ensure pod to pod communication will not be masquerade
// kindnetd will also write a templated cni config supplied with PodCIDR
//
// input envs:
// - HOST_IP: should be populated by downward API
// - POD_IP: should be populated by downward API
// - CNI_CONFIG_TEMPLATE: the cni .conflist template, run with {{ .PodCIDR }}

// TODO: improve logging & error handling

// IPFamily defines kindnet networking operating model
type IPFamily string

const (
	// IPv4Family sets IPFamily to ipv4
	IPv4Family IPFamily = "ipv4"
	// IPv6Family sets IPFamily to ipv6
	IPv6Family IPFamily = "ipv6"
	// DualStackFamily sets ClusterIPFamily to DualStack
	DualStackFamily IPFamily = "DualStack"
)

func main() {
	// enable logging
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "true")
	flag.Parse()

	// create a Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// obtain the host and pod ip addresses
	hostIP, podIP := os.Getenv("HOST_IP"), os.Getenv("POD_IP")
	klog.Infof("hostIP = %s\npodIP = %s\n", hostIP, podIP)
	if hostIP != podIP {
		panic(fmt.Sprintf(
			"hostIP(= %q) != podIP(= %q) but must be running with host network: ",
			hostIP, podIP,
		))
	}

	// used to track if the cni config inputs changed and write the config
	cniConfigWriter := &CNIConfigWriter{
		path: cniConfigPath,
	}

	// enforce ip masquerade rules
	noMaskIPv4Subnets, noMaskIPv6Subnets := getNoMasqueradeSubnets(clientset)
	// detect the cluster IP family based on the Cluster CIDR akka PodSubnet
	var ipFamily IPFamily
	if len(noMaskIPv4Subnets) > 0 && len(noMaskIPv6Subnets) > 0 {
		ipFamily = DualStackFamily
	} else if len(noMaskIPv6Subnets) > 0 {
		ipFamily = IPv6Family
	} else if len(noMaskIPv4Subnets) > 0 {
		ipFamily = IPv4Family
	} else {
		panic(fmt.Sprint("Cluster CIDR is not defined"))
	}
	klog.Infof("kindnetd IP family: %q", ipFamily)

	// create an ipMasqAgent for IPv4
	if len(noMaskIPv4Subnets) > 0 {
		klog.Infof("noMask IPv4 subnets: %v", noMaskIPv4Subnets)
		masqAgentIPv4, err := NewIPMasqAgent(false, noMaskIPv4Subnets)
		if err != nil {
			panic(err.Error())
		}
		go func() {
			if err := masqAgentIPv4.SyncRulesForever(time.Second * 60); err != nil {
				panic(err)
			}
		}()
	}

	// create an ipMasqAgent for IPv6
	if len(noMaskIPv6Subnets) > 0 {
		klog.Infof("noMask IPv6 subnets: %v", noMaskIPv6Subnets)
		masqAgentIPv6, err := NewIPMasqAgent(true, noMaskIPv6Subnets)
		if err != nil {
			panic(err.Error())
		}

		go func() {
			if err := masqAgentIPv6.SyncRulesForever(time.Second * 60); err != nil {
				panic(err)
			}
		}()
	}

	// setup nodes reconcile function, closes over arguments
	reconcileNodes := makeNodesReconciler(cniConfigWriter, hostIP, ipFamily, clientset)

	// main control loop
	for {
		// Gets the Nodes information from the API
		// TODO: use a proper controller instead
		var nodes *corev1.NodeList
		var err error
		for i := 0; i < 5; i++ {
			nodes, err = clientset.CoreV1().Nodes().List(metav1.ListOptions{})
			if err == nil {
				break
			}
			klog.Infof("Failed to get nodes, retrying after error: %v", err)
			time.Sleep(time.Second * time.Duration(i))
		}
		if err != nil {
			panic("Reached maximum retries obtaining node list: " + err.Error())
		}

		// reconcile the nodes with retries
		for i := 0; i < 5; i++ {
			err = reconcileNodes(nodes)
			if err == nil {
				break
			}
			klog.Infof("Failed to reconcile routes, retrying after error: %v", err)
			time.Sleep(time.Second * time.Duration(i))
		}
		if err != nil {
			panic("Maximum retries reconciling node routes: " + err.Error())
		}

		// rate limit
		time.Sleep(10 * time.Second)
	}
}

// nodeNodesReconciler returns a reconciliation func for nodes
func makeNodesReconciler(cniConfig *CNIConfigWriter, hostIP string, ipFamily IPFamily, clientset *kubernetes.Clientset) func(*corev1.NodeList) error {
	// reconciles a node
	reconcileNode := func(node corev1.Node) error {
		// first get this node's IPs
		// we don't support more than one IP address per IP family for simplification
		nodeIPs := internalIPs(node)
		klog.Infof("Handling node with IPs: %v\n", nodeIPs)
		// This is our node. We don't need to add routes, but we might need to
		// update the cni config and "annotate" our external IPs
		if nodeIPs.Has(hostIP) {
			klog.Info("handling current node\n")
			// compute the current cni config inputs
			if err := cniConfig.Write(
				ComputeCNIConfigInputs(node),
			); err != nil {
				return err
			}
			if ipFamily == DualStackFamily {
				// update external IPs node annotations until #42125 is fixed
				// TODO: https://github.com/kubernetes/kubernetes/issues/42125
				// in KIND we know that the outer interface is eth0 to find the external ips
				// but we can be smarter and find the interface based on the HOST_IP
				hostExternalIPs := getHostExternalIPs(hostIP)
				// update the node
				klog.Infof("Update node with IPs: %v\n", hostExternalIPs)
				patchString := fmt.Sprintf(`{"metadata": {"annotations": {"kind.x-k8s.io.kindnet/addresses": "%s"}}}`, hostExternalIPs)
				patchBytes := []byte(patchString)
				if _, err := clientset.CoreV1().Nodes().Patch(node.Name, types.MergePatchType, patchBytes); err != nil {
					return fmt.Errorf("failed to patch node: %v", err)
				}
			}
			// we're done handling this node
			return nil
		}

		// This is another node. Add routes to the POD subnets in the other nodes
		// don't do anything unless there is a PodCIDR
		var podCIDRs []string
		if ipFamily == DualStackFamily {
			podCIDRs = node.Spec.PodCIDRs
		} else {
			podCIDRs = []string{node.Spec.PodCIDR}
		}
		if len(podCIDRs) == 0 {
			fmt.Printf("Node %v has no CIDR, ignoring\n", node.Name)
			return nil
		}
		klog.Infof("Node %v has CIDR %s \n", node.Name, podCIDRs)
		podCIDRsv4, podCIDRsv6 := splitCIDRs(podCIDRs)

		// obtain the PodCIDR gateway
		var nodeIPv4, nodeIPv6 string
		for _, ip := range nodeIPs.List() {
			if utilsnet.IsIPv6String(ip) {
				nodeIPv6 = ip
			} else {
				nodeIPv4 = ip
			}
		}

		if nodeIPv4 != "" && len(podCIDRsv4) > 0 {
			if err := syncRoute(nodeIPv4, podCIDRsv4); err != nil {
				return err
			}
		}
		if nodeIPv6 != "" && len(podCIDRsv6) > 0 {
			if err := syncRoute(nodeIPv6, podCIDRsv6); err != nil {
				return err
			}
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

// internalIPs returns the internal IP address for node
func internalIPs(node corev1.Node) sets.String {
	ips := sets.NewString()
	// check the node.Status.Addresses
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			ips.Insert(address.Address)
		}
	}
	// check the node.annotations.Internal.Addresses
	for _, address := range strings.Split(node.Annotations["kind.x-k8s.io.kindnet/addresses"], ",") {
		ips.Insert(address)
	}
	return ips
}

// getHostExternalIPs return a comma separated list with the
// node external IP addresses
func getHostExternalIPs(ip string) string {
	return getInterfaceIPs(interfaceByAddress(ip))
}

// interfaceByAddress return the name of the interface
// that hosts a given IP address
func interfaceByAddress(address string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err.Error())
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err.Error())
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPAddr:
				ip = v.IP
			case *net.IPNet:
				ip = v.IP
			default:
				continue
			}
			if ip.String() == address {
				return i.Name
			}

		}
	}
	return ""
}

// getInterfaceIPs returns an array with all the global addresses
// of the interfaces passed as a parameter
func getInterfaceIPs(ifazName string) string {
	var ips string

	ifaz, err := net.InterfaceByName(ifazName)
	if err != nil {
		panic(err.Error())
	}

	addrs, err := ifaz.Addrs()
	if err != nil {
		panic(err.Error())
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPAddr:
			ip = v.IP
		case *net.IPNet:
			ip = v.IP
		default:
			continue
		}
		if ip.IsGlobalUnicast() {
			ips = ips + "," + ip.String()
		}

	}
	return strings.Trim(ips, ",")
}
