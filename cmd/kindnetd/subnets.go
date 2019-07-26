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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/sets"
)

type subnets struct {
	pods sets.String
	services sets.String
}

func newSubnets() *subnets {
	return &subnets{
		pods: sets.NewString(),
		services: sets.NewString(),
	}
}

func (s *subnets) All() []string {
	return s.pods.Union(s.services).List()
}


func getNoMasqueradeSubnets(clientset *kubernetes.Clientset) []string {
	s := newSubnets()

	// check for environment variables (legacy)
	podSubnetEnv := os.Getenv("POD_SUBNET")
	if podSubnetEnv != "" {
		s.pods.Insert(podSubnetEnv)
	}
	serviceSubnetEnv := os.Getenv("SERVICE_SUBNET")
	if serviceSubnetEnv != "" {
		s.services.Insert(serviceSubnetEnv)
	}

	// try getting from kubeadm config
	kubeadmSubnets := getKubeadmSubnets(clientset)
	s.pods = s.pods.Union(kubeadmSubnets.pods)
	s.services = s.services.Union(kubeadmSubnets.services)
	
	// TODO: we have other fallback options for completeness including:
	// - kube-apiserver flags
	// - component config
	// - defaults

	return s.All()
}




const (
	kubeadmServiceCIDRRegex = `\s*serviceSubnet: (.*)\n`
	kubeadmClusterCIDRRegex = `\s*podSubnet: (.*)\n`
)
var (
	kubeadmServiceCIDRRegexp = regexp.MustCompile(kubeadmServiceCIDRRegex)
	kubeadmClusterCIDRRegexp = regexp.MustCompile(kubeadmClusterCIDRRegex)
)

func getKubeadmSubnets(clientset *kubernetes.Clientset) *subnets {
	s := newSubnets()
	configMap, err := clientset.CoreV1().ConfigMaps("kube-system").Get("kubeadm-config", metav1.GetOptions{})
	if err != nil {
		println("ERROR: "+err.Error())
		return s
	}
	for _, data := range configMap.Data {
		matches := kubeadmServiceCIDRRegexp.FindAllStringSubmatch(data, 1)
		if len(matches) > 0 {
			println("MATCH SERVICE: " + matches[0][1])
			s.services.Insert(matches[0][1])
		}
		matches = kubeadmClusterCIDRRegexp.FindAllStringSubmatch(data, 1)
		if len(matches) > 0 {
			println("MATCH POD: " + matches[0][1])
			s.pods.Insert(matches[0][1])
		}
	}
	return s
}

