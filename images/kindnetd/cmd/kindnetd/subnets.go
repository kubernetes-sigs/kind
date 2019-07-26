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
	"os"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/net"
)

const (
	kubeadmClusterCIDRRegex   = `\s+podSubnet: (.*)\n`
	kubeproxyClusterCIDRRegex = `\s+clusterCIDR: (.*)\n`
)

type kubeSubnets struct {
	regex     string
	configmap string
	namespace string
}

func newKubeSubnets(regex, configmap, namespace string) *kubeSubnets {
	return &kubeSubnets{
		regex:     regex,
		configmap: configmap,
		namespace: namespace,
	}
}

func (ks *kubeSubnets) Get(clientset *kubernetes.Clientset) (string, error) {
	cidrRegexp := regexp.MustCompile(ks.regex)
	configMap, err := clientset.CoreV1().ConfigMaps(ks.namespace).Get(ks.configmap, metav1.GetOptions{})
	if err != nil {
		println("ERROR: " + err.Error())
		return "", err
	}
	for _, data := range configMap.Data {
		matches := cidrRegexp.FindStringSubmatch(data)
		if len(matches) > 0 {
			println("MATCH POD: " + matches[1])
			return matches[1], nil
		}
	}
	return "", nil
}

// getNoMasqueradeSubnets tries to get the POD networks subnets to not Masquerade them
// It returns an array of strings with the Cluster CIDR subnets
// It can only obtain the POD subnet parameter from one place for consistency
// The order is:
// 1. POD_SUBNET environment variables
// 2. Pod subnet value in kubeadm configmap
// 3. Cluster CIDR value in kube-proxy configmap
func getNoMasqueradeSubnets(clientset *kubernetes.Clientset) ([]string, []string) {
	// check for environment variables (legacy)
	podSubnetEnv := os.Getenv("POD_SUBNET")
	if podSubnetEnv != "" {
		podSubnetEnv = strings.TrimSpace(podSubnetEnv)
		return splitCIDRs(strings.Split(podSubnetEnv, ","))
	}

	// try getting from kubeadm configmap
	kubeadmSubnets := newKubeSubnets(kubeadmClusterCIDRRegex, "kubeadm-config", "kube-system")
	podSubnetKubeadm, _ := kubeadmSubnets.Get(clientset)
	if podSubnetKubeadm != "" {
		podSubnetKubeadm = strings.TrimSpace(podSubnetKubeadm)
		return splitCIDRs(strings.Split(podSubnetKubeadm, ","))
	}

	// try getting from kubeproxy configmap
	kubeproxySubnets := newKubeSubnets(kubeproxyClusterCIDRRegex, "kube-proxy", "kube-system")
	podSubnetKubeproxy, _ := kubeproxySubnets.Get(clientset)
	if podSubnetKubeproxy != "" {
		podSubnetKubeproxy = strings.TrimSpace(podSubnetKubeproxy)
		return splitCIDRs(strings.Split(podSubnetKubeproxy, ","))

	}

	// TODO: we have other fallback options for completeness including:
	// - kube-apiserver flags
	// - component config
	// - defaults
	return nil, nil
}

// splitCIDRs given a slice of strings with CIDRS it returns 2 slice of strings per IP family
func splitCIDRs(cidrs []string) ([]string, []string) {
	var v4subnets, v6subnets []string
	for _, subnet := range cidrs {
		if net.IsIPv6CIDRString(subnet) {
			v6subnets = append(v6subnets, subnet)
		} else {
			v4subnets = append(v4subnets, subnet)
		}
	}
	return v4subnets, v6subnets
}
