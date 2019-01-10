/*
Copyright 2018 The Kubernetes Authors.

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

package config

// IsControlPlane returns true if the node hosts a control plane instance
// NB. in single node clusters, control-plane nodes act also as a worker nodes
func (n *Node) IsControlPlane() bool {
	return n.Role == ControlPlaneRole
}

// IsWorker returns true if the node hosts a worker instance
func (n *Node) IsWorker() bool {
	return n.Role == WorkerRole
}

// IsExternalEtcd returns true if the node hosts an external etcd member
func (n *Node) IsExternalEtcd() bool {
	return n.Role == ExternalEtcdRole
}

// IsExternalLoadBalancer returns true if the node hosts an external load balancer
func (n *Node) IsExternalLoadBalancer() bool {
	return n.Role == ExternalLoadBalancerRole
}
