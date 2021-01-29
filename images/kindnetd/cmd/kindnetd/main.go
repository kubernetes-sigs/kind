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
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	utilsnet "k8s.io/utils/net"
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

	// obtain the host and pod ip addresses
	// if both ips are different we are not using the host network
	hostIP, podIP := os.Getenv("HOST_IP"), os.Getenv("POD_IP")
	klog.Infof("hostIP = %s\npodIP = %s\n", hostIP, podIP)
	if hostIP != podIP {
		panic(fmt.Sprintf(
			"hostIP(= %q) != podIP(= %q) but must be running with host network: ",
			hostIP, podIP,
		))
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
	// TODO: dual stack...?
	masqAgent, err := NewIPMasqAgent(utilsnet.IsIPv6String(hostIP), []string{os.Getenv("POD_SUBNET")})
	if err != nil {
		panic(err.Error())
	}
	// run the masqAgent and panic if is not able to install the rules to no masquerade the pod to pod traffic
	go func() {
		if err := masqAgent.SyncRulesForever(time.Second * 60); err != nil {
			panic(err.Error())
		}
	}()

	// setup nodes reconcile function, closes over arguments
	reconcileNodes := makeNodesReconciler(cniConfigWriter, hostIP)

	// main control loop
	for {
		// Gets the Nodes information from the API
		// TODO: use a proper controller instead
		var nodes *corev1.NodeList
		var err error
		for i := 0; i < 5; i++ {
			nodes, err = clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
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
func makeNodesReconciler(cniConfig *CNIConfigWriter, hostIP string) func(*corev1.NodeList) error {
	// reconciles a node
	reconcileNode := func(node corev1.Node) error {
		// first get this node's IP
		nodeIP := internalIP(node)
		if nodeIP == "" {
			klog.Infof("Node %v has no Internal IP, ignoring\n", node.Name)
			return nil
		}

		// don't do anything unless there is a PodCIDR
		podCIDR := node.Spec.PodCIDR
		if podCIDR == "" {
			klog.Infof("Node %v has no CIDR, ignoring\n", node.Name)
			return nil
		}

		// This is our node. We don't need to add routes, but we might need to
		// update the cni config.
		if nodeIP == hostIP {
			klog.Infof("handling current node\n")
			// compute the current cni config inputs
			if err := cniConfig.Write(
				ComputeCNIConfigInputs(node),
			); err != nil {
				return err
			}
			// we're done handling this node
			return nil
		}

		klog.Infof("Handling node with IP: %s\n", nodeIP)
		klog.Infof("Node %v has CIDR %s \n", node.Name, podCIDR)
		if err := syncRoute(nodeIP, podCIDR); err != nil {
			return err
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
