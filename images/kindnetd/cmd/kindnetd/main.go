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
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kube-network-policies/pkg/networkpolicy"
)

const (
	probeTCPtimeout = 1 * time.Second
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
// - CONTROL_PLANE_ENDPOINT: control-plane endpoint format host:port

// TODO: improve logging & error handling

// IPFamily defines kindnet networking operating model
type IPFamily string

const (
	// IPv4Family sets IPFamily to ipv4
	IPv4Family IPFamily = "ipv4"
	// IPv6Family sets IPFamily to ipv6
	IPv6Family IPFamily = "ipv6"
	// DualStackFamily sets ClusterIPFamily to DualStack
	DualStackFamily IPFamily = "dualstack"
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
	// use protobuf to improve performance
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	// override the internal apiserver endpoint to avoid
	// waiting for kube-proxy to install the services rules.
	// If the endpoint is not reachable, fallback the internal endpoint
	controlPlaneEndpoint := os.Getenv("CONTROL_PLANE_ENDPOINT")
	if controlPlaneEndpoint != "" {
		// check that the apiserver is reachable before continue
		// to fail fast and avoid waiting until the client operations timeout
		var ok bool
		for i := 0; i < 5; i++ {
			ok = probeTCP(controlPlaneEndpoint, probeTCPtimeout)
			if ok {
				config.Host = "https://" + controlPlaneEndpoint
				break
			}
			klog.Infof("apiserver not reachable, attempt %d ... retrying", i)
			time.Sleep(time.Second * time.Duration(i))
		}
	}
	// create the clientset to connect the apiserver
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	klog.Infof("connected to apiserver: %s", config.Host)

	// trap Ctrl+C and call cancel on the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Enable signal handler
	signalCh := make(chan os.Signal, 2)
	defer func() {
		close(signalCh)
		cancel()
	}()
	signal.Notify(signalCh, os.Interrupt, unix.SIGINT)

	go func() {
		select {
		case <-signalCh:
			klog.Infof("Exiting: received signal")
			cancel()
		case <-ctx.Done():
		}
	}()

	informersFactory := informers.NewSharedInformerFactory(clientset, 0)
	nodeInformer := informersFactory.Core().V1().Nodes()
	nodeLister := nodeInformer.Lister()

	// obtain the host and pod ip addresses
	// if both ips are different we are not using the host network
	hostIP, podIP := os.Getenv("HOST_IP"), os.Getenv("POD_IP")
	klog.Infof("hostIP = %s\npodIP = %s\n", hostIP, podIP)
	if hostIP != podIP {
		klog.Warningf(
			"hostIP(= %q) != podIP(= %q) but must be running with host network: ",
			hostIP, podIP,
		)
	}

	mtu, err := computeBridgeMTU()
	klog.Infof("setting mtu %d for CNI \n", mtu)
	if err != nil {
		klog.Infof("Failed to get MTU size from interface eth0, using kernel default MTU size error:%v", err)
	}
	// used to track if the cni config inputs changed and write the config
	cniConfigWriter := &CNIConfigWriter{
		path: cniConfigPath,
		mtu:  mtu,
	}

	// enforce ip masquerade rules
	podSubnetEnv := os.Getenv("POD_SUBNET")
	if podSubnetEnv == "" {
		panic("missing environment variable POD_SUBNET")
	}
	podSubnetEnv = strings.TrimSpace(podSubnetEnv)
	podSubnets := strings.Split(podSubnetEnv, ",")
	clusterIPv4Subnets, clusterIPv6Subnets := splitCIDRs(podSubnets)

	// detect the cluster IP family based on the Cluster CIDR aka PodSubnet
	var ipFamily IPFamily
	if len(clusterIPv4Subnets) > 0 && len(clusterIPv6Subnets) > 0 {
		ipFamily = DualStackFamily
	} else if len(clusterIPv6Subnets) > 0 {
		ipFamily = IPv6Family
	} else if len(clusterIPv4Subnets) > 0 {
		ipFamily = IPv4Family
	} else {
		panic(fmt.Sprintf("podSubnets ClusterCIDR/Pod_Subnet: %v", podSubnetEnv))
	}
	klog.Infof("kindnetd IP family: %q", ipFamily)

	// create an ipMasqAgent for IPv4
	if len(clusterIPv4Subnets) > 0 {
		klog.Infof("noMask IPv4 subnets: %v", clusterIPv4Subnets)
		masqAgentIPv4, err := NewIPMasqAgent(false, clusterIPv4Subnets)
		if err != nil {
			panic(err.Error())
		}
		go func() {
			if err := masqAgentIPv4.SyncRulesForever(ctx, time.Second*60); err != nil {
				panic(err)
			}
		}()
	}

	// create an ipMasqAgent for IPv6
	if len(clusterIPv6Subnets) > 0 {
		klog.Infof("noMask IPv6 subnets: %v", clusterIPv6Subnets)
		masqAgentIPv6, err := NewIPMasqAgent(true, clusterIPv6Subnets)
		if err != nil {
			panic(err.Error())
		}

		go func() {
			if err := masqAgentIPv6.SyncRulesForever(ctx, time.Second*60); err != nil {
				panic(err)
			}
		}()
	}

	// setup nodes reconcile function, closes over arguments
	reconcileNodes := makeNodesReconciler(cniConfigWriter, hostIP, ipFamily)

	// network policies

	// on kind nodes the hostname matches the node name
	nodeName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("couldn't determine hostname: %v", err)
	}

	cfg := networkpolicy.Config{
		FailOpen: true,
		QueueID:  100,
		NodeName: nodeName,
	}

	networkPolicyController, err := networkpolicy.NewController(
		clientset,
		informersFactory.Networking().V1().NetworkPolicies(),
		informersFactory.Core().V1().Namespaces(),
		informersFactory.Core().V1().Pods(),
		nodeInformer,
		nil,
		nil,
		nil,
		cfg)
	if err != nil {
		klog.Infof("Error creating network policy controller: %v, skipping network policies", err)
	} else {
		go func() {
			_ = networkPolicyController.Run(ctx)
		}()
	}

	// main control loop
	informersFactory.Start(ctx.Done())
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		var nodes []*corev1.Node
		var err error
		for i := 0; i < 5; i++ {
			nodes, err = nodeLister.List(labels.Everything())
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
		select {
		case <-ctx.Done():
			// grace period to cleanup resources
			time.Sleep(1 * time.Second)
			return
		case <-ticker.C:
		}
	}
}

// nodeNodesReconciler returns a reconciliation func for nodes
func makeNodesReconciler(cniConfig *CNIConfigWriter, hostIP string, ipFamily IPFamily) func([]*corev1.Node) error {
	// reconciles a node
	reconcileNode := func(node *corev1.Node) error {
		// first get this node's IPs
		// we don't support more than one IP address per IP family for simplification
		nodeIPs := internalIPs(node)
		klog.Infof("Handling node with IPs: %v\n", nodeIPs)
		// This is our node. We don't need to add routes,
		// but we might need to update the cni config
		if nodeIPs.Has(hostIP) {
			klog.Info("handling current node\n")
			// compute the current cni config inputs
			if err := cniConfig.Write(
				ComputeCNIConfigInputs(node),
			); err != nil {
				return err
			}
			// we're done handling this node
			return nil
		}

		// This is another node. Add routes to the POD subnets in the other nodes
		// don't do anything unless there is a non-empty PodCIDR
		var podCIDRs []string
		if ipFamily == DualStackFamily {
			podCIDRs = node.Spec.PodCIDRs
		} else if node.Spec.PodCIDR != "" {
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
		for _, ip := range nodeIPs.UnsortedList() {
			if isIPv6String(ip) {
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
	return func(nodes []*corev1.Node) error {
		for _, node := range nodes {
			if err := reconcileNode(node); err != nil {
				return err
			}
		}
		return nil
	}
}

// internalIPs returns the internal IP addresses for node
func internalIPs(node *corev1.Node) sets.Set[string] {
	ips := sets.New[string]()
	// check the node.Status.Addresses
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			ips.Insert(address.Address)
		}
	}
	return ips
}

// splitCIDRs given a slice of strings with CIDRs it returns 2 slice of strings per IP family
// The order returned is always v4 v6
func splitCIDRs(cidrs []string) ([]string, []string) {
	var v4subnets, v6subnets []string
	for _, subnet := range cidrs {
		if isIPv6CIDRString(subnet) {
			v6subnets = append(v6subnets, subnet)
		} else {
			v4subnets = append(v4subnets, subnet)
		}
	}
	return v4subnets, v6subnets
}

// Modified from agnhost connect command in k/k
// https://github.com/kubernetes/kubernetes/blob/c241a237f9a635286c76c20d07b103a663b1cfa4/test/images/agnhost/connect/connect.go#L66
func probeTCP(address string, timeout time.Duration) bool {
	klog.Infof("probe TCP address %s", address)
	if _, err := net.ResolveTCPAddr("tcp", address); err != nil {
		klog.Warningf("DNS problem %s: %v", address, err)
		return false
	}

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err == nil {
		conn.Close()
		return true
	}
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Timeout() {
			klog.Warningf("TIMEOUT %s", address)
		} else if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
			if syscallErr.Err == syscall.ECONNREFUSED {
				klog.Warningf("REFUSED %s", address)
			}
		}
		return false
	}

	klog.Warningf("OTHER %s: %v", address, err)
	return false
}

// isIPv6String returns if ip is IPv6.
func isIPv6String(ip string) bool {
	netIP := net.ParseIP(ip)
	return netIP != nil && netIP.To4() == nil
}

// isIPv6CIDRString returns if cidr is IPv6.
// This assumes cidr is a valid CIDR.
func isIPv6CIDRString(cidr string) bool {
	ip, _, _ := net.ParseCIDR(cidr)
	return ip != nil && ip.To4() == nil
}
