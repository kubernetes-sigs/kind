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

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
)

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

// ProvisioningOrder returns the provisioning order for nodes, that
// should be defined according to the assigned NodeRole
func (n *Node) ProvisioningOrder() int {
	switch n.Role {
	// External dependencies should be provisioned first; we are defining an arbitrary
	// precedence between etcd and load balancer in order to get predictable/repeatable results
	case ExternalEtcdRole:
		return 1
	case ExternalLoadBalancerRole:
		return 2
	// Then control plane nodes
	case ControlPlaneRole:
		return 3
	// Finally workers
	case WorkerRole:
		return 4
	default:
		return 99
	}
}

// Len of the NodeList.
// It is required for making NodeList sortable.
func (t ReplicaList) Len() int {
	return len(t)
}

// Less return the lower between two elements of the NodeList, where the
// lower element should be provisioned before the other.
// It is required for making NodeList sortable.
func (t ReplicaList) Less(i, j int) bool {
	return t[i].ProvisioningOrder() < t[j].ProvisioningOrder() ||
		// In case of same provisioning order, the name is used to get predictable/repeatable results
		(t[i].ProvisioningOrder() == t[j].ProvisioningOrder() && t[i].Name < t[j].Name)
}

// Swap two elements of the NodeList.
// It is required for making NodeList sortable.
func (t ReplicaList) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// DeriveInfo populates DerivedConfig info starting
// from the current list on Nodes
func (c *Config) DeriveInfo() error {
	for _, n := range c.Nodes {
		if err := c.Add(&n); err != nil {
			return err
		}
	}
	return nil
}

// Add a Node to the `kind` cluster, generating requested node replicas
// and assigning a unique node name to each replica.
func (c *Config) Add(node *Node) error {

	// Creates the list of node replicas
	expectedReplicas := 1
	if node.Replicas != nil {
		expectedReplicas = int(*node.Replicas)
	}

	replicas := ReplicaList{}
	for i := 1; i <= expectedReplicas; i++ {
		replica := &NodeReplica{
			Node: *node.DeepCopy(),
		}
		replica.Replicas = nil // resetting the replicas number for each replica to default (1)

		replicas = append(replicas, replica)
	}

	// adds replica to the config unpdating derivedConfigData
	for _, replica := range replicas {

		// adds the replica to the list of nodes
		c.allReplicas = append(c.allReplicas, replica)

		// list of nodes with control plane role
		if replica.IsControlPlane() {
			// assign selected name for control plane node
			replica.Name = "control-plane"
			// stores the node in derivedConfigData
			c.controlPlanes = append(c.controlPlanes, replica)
		}

		// list of nodes with worker role
		if replica.IsWorker() {
			// assign selected name for worker node
			replica.Name = "worker"
			// stores the node in derivedConfigData
			c.workers = append(c.workers, replica)
		}

		// node with external etcd role
		if replica.IsExternalEtcd() {
			if c.externalEtcd != nil {
				return errors.Errorf("invalid config. there are two nodes with role %q", ExternalEtcdRole)
			}
			// assign selected name for etcd node
			replica.Name = "etcd"
			// stores the node in derivedConfigData
			c.externalEtcd = replica
		}

		// node with external load balancer role
		if replica.IsExternalLoadBalancer() {
			if c.externalLoadBalancer != nil {
				return errors.Errorf("invalid config. there are two nodes with role %q", ExternalLoadBalancerRole)
			}
			// assign selected name for load balancer node
			replica.Name = "lb"
			// stores the node in derivedConfigData
			c.externalLoadBalancer = replica
		}

	}

	// if more than one control plane node exists, fixes names to get a progressive index
	if len(c.controlPlanes) > 1 {
		for i, n := range c.controlPlanes {
			n.Name = fmt.Sprintf("%s%d", "control-plane", i+1)
		}
	}

	// if more than one worker node exists, fixes names to get a progressive index
	if len(c.workers) > 1 {
		for i, n := range c.workers {
			n.Name = fmt.Sprintf("%s%d", "worker", i+1)
		}
	}

	// ensure the list of nodes is ordered.
	// the ordering is key for getting a consistent and predictable behaviour
	// when provisioning nodes and when executing actions on nodes
	sort.Sort(c.allReplicas)

	return nil
}

// AllReplicas returns all the node replicas defined in the `kind` Config.
func (c *Config) AllReplicas() ReplicaList {
	return c.allReplicas
}

// ControlPlanes returns all the nodes with control-plane role
func (c *Config) ControlPlanes() ReplicaList {
	return c.controlPlanes
}

// BootStrapControlPlane returns the first node with control-plane role
// This is the node where kubeadm init will be executed.
func (c *Config) BootStrapControlPlane() *NodeReplica {
	if len(c.controlPlanes) == 0 {
		return nil
	}
	return c.controlPlanes[0]
}

// SecondaryControlPlanes returns all the nodes with control-plane role
// except the BootStrapControlPlane node, if any,
func (c *Config) SecondaryControlPlanes() ReplicaList {
	if len(c.controlPlanes) <= 1 {
		return nil
	}
	return c.controlPlanes[1:]
}

// Workers returns all the nodes with Worker role, if any
func (c *Config) Workers() ReplicaList {
	return c.workers
}

// ExternalEtcd returns the node with external-etcd role, if defined
func (c *Config) ExternalEtcd() *NodeReplica {
	return c.externalEtcd
}

// ExternalLoadBalancer returns the node with external-load-balancer role, if defined
func (c *Config) ExternalLoadBalancer() *NodeReplica {
	return c.externalLoadBalancer
}
