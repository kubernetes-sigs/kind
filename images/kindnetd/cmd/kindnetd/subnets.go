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
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type subnets struct {
	pods     sets.String
	services sets.String
}

func newSubnets() *subnets {
	return &subnets{
		pods:     sets.NewString(),
		services: sets.NewString(),
	}
}

func (s *subnets) All() []string {
	return s.pods.Union(s.services).List()
}

func getNoMasqueradeSubnets(clientset *kubernetes.Clientset) []string {
	s := newSubnets()

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
	kubeadmServiceCIDRRegex = `(?m)^\s*serviceSubnet: (.*)\n`
	kubeadmClusterCIDRRegex = `(?m)^\s*podSubnet: (.*)\n`
)

var (
	kubeadmServiceCIDRRegexp = regexp.MustCompile(kubeadmServiceCIDRRegex)
	kubeadmClusterCIDRRegexp = regexp.MustCompile(kubeadmClusterCIDRRegex)
)

func getKubeadmSubnets(clientset *kubernetes.Clientset) *subnets {
	s := newSubnets()
	configMap, err := clientset.CoreV1().ConfigMaps("kube-system").Get("kubeadm-config", metav1.GetOptions{})
	if err != nil {
		fmt.Println("error getting kubeadm config: " + err.Error())
		return s
	}
	for _, data := range configMap.Data {
		// ensure data is yaml formatted and does not contain comments
		cleaned, err := yamlify([]byte(data))
		if err != nil {
			fmt.Printf("Error parsing kubeadm configmap: %v\n", err)
			continue
		}
		matches := kubeadmServiceCIDRRegexp.FindAllSubmatch(cleaned, 1)
		if len(matches) > 0 {
			s.services.Insert(string(matches[0][1]))
		}
		matches = kubeadmClusterCIDRRegexp.FindAllSubmatch(cleaned, 1)
		if len(matches) > 0 {
			s.pods.Insert(string(matches[0][1]))
		}
	}
	return s
}

func yamlify(yamlOrJSON []byte) ([]byte, error) {
	var unstructured interface{}
	if err := yaml.Unmarshal(yamlOrJSON, &unstructured); err != nil {
		return nil, err
	}
	return yaml.Marshal(&unstructured)
}
