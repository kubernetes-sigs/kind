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

package create

import (
	"fmt"
	"sort"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/util"

	"github.com/pkg/errors"
)

// DerivedConfig contains config-like data computed from pkg/cluster/config.Config
// namely, it contains lists of NodeReplicas to be created based on the config
type DerivedConfig struct {
	// allReplicas constains the list of node replicas defined in the `kind` Config
	allReplicas ReplicaList
	// controlPlanes contains the subset of node replicas with control-plane role
	controlPlanes ReplicaList
	// workers contains the subset of node replicas with worker role, if any
	workers ReplicaList
	// externalEtcd contains the node replica with external-etcd role, if defined
	// TODO(fabriziopandini): eventually in future we would like to support
	// external etcd clusters with more than one member
	externalEtcd *NodeReplica
	// externalLoadBalancer contains the node replica with external-load-balancer role, if defined
	externalLoadBalancer *NodeReplica
}

// NodeReplica defines a `kind` config Node that is geneated by creating a replicas for a node
// This attribute exists only in the internal config version and is meant
// to simplify the usage of the config in the code base.
type NodeReplica struct {
	// Node contains settings for the node in the `kind` Config.
	// please note that the Replicas number is alway set to nil.
	config.Node

	// Name contains the unique name assigned to the node while generating the replica
	Name string
}

// ReplicaList defines a list of NodeReplicas in the `kind` Config
// This attribute exists only in the internal config version and is meant
// to simplify the usage of the config in the code base.
type ReplicaList []*NodeReplica

// Validate validates that the configuration is possible to create
func (d *DerivedConfig) Validate() error {
	errs := []error{}

	// There should be at least one control plane
	if d.BootStrapControlPlane() == nil {
		errs = append(errs, errors.Errorf("please add at least one node with role %q", config.ControlPlaneRole))
	}
	// There should be one load balancer if more than one control plane exists in the cluster
	if len(d.ControlPlanes()) > 1 && d.ExternalLoadBalancer() == nil {
		errs = append(errs, errors.Errorf("please add a node with role %s because in the cluster there are more than one node with role %s",
			config.ExternalLoadBalancerRole, config.ControlPlaneRole))
	}

	// TODO(fabrizio pandini): this check is temporary / WIP
	// kind v1alpha config fully supports multi nodes, but the cluster creation logic implemented in
	// pkg/cluster/contex.go does it only partially (yet).
	// As soon as external etcd is implemented in pkg/cluster, this should go away
	if d.ExternalEtcd() != nil {
		errs = append(errs, errors.New("multi node support is still a work in progress, currently external etcd node is not supported"))
	}

	if len(errs) > 0 {
		return util.NewErrors(errs)
	}
	return nil
}

// ProvisioningOrder returns the provisioning order for nodes, that
// should be defined according to the assigned NodeRole
func (n *NodeReplica) ProvisioningOrder() int {
	switch n.Role {
	// External dependencies should be provisioned first; we are defining an arbitrary
	// precedence between etcd and load balancer in order to get predictable/repeatable results
	case config.ExternalEtcdRole:
		return 1
	case config.ExternalLoadBalancerRole:
		return 2
	// Then control plane nodes
	case config.ControlPlaneRole:
		return 3
	// Finally workers
	case config.WorkerRole:
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

// Derive populates DerivedConfig info starting
// from the current list on Nodes
func Derive(c *config.Config) (*DerivedConfig, error) {
	d := &DerivedConfig{}

	for _, n := range c.Nodes {
		if err := d.Add(&n); err != nil {
			return nil, err
		}
	}

	return d, nil
}

// Add a Node to the `kind` cluster, generating requested node replicas
// and assigning a unique node name to each replica.
func (d *DerivedConfig) Add(node *config.Node) error {

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

	// adds replica to the config updating derivedConfig
	for _, replica := range replicas {

		// adds the replica to the list of nodes
		d.allReplicas = append(d.allReplicas, replica)

		// list of nodes with control plane role
		if replica.Role == config.ControlPlaneRole {
			// assign selected name for control plane node
			replica.Name = "control-plane"
			// stores the node in derivedConfig
			d.controlPlanes = append(d.controlPlanes, replica)
		}

		// list of nodes with worker role
		if replica.Role == config.ControlPlaneRole {
			// assign selected name for worker node
			replica.Name = "worker"
			// stores the node in derivedConfig
			d.workers = append(d.workers, replica)
		}

		// node with external etcd role
		if replica.Role == config.ExternalEtcdRole {
			if d.externalEtcd != nil {
				return errors.Errorf("invalid config. there are two nodes with role %q", config.ExternalEtcdRole)
			}
			// assign selected name for etcd node
			replica.Name = "etcd"
			// stores the node in derivedConfig
			d.externalEtcd = replica
		}

		// node with external load balancer role
		if replica.Role == config.ExternalLoadBalancerRole {
			if d.externalLoadBalancer != nil {
				return errors.Errorf("invalid config. there are two nodes with role %q", config.ExternalLoadBalancerRole)
			}
			// assign selected name for load balancer node
			replica.Name = "lb"
			// stores the node in derivedConfig
			d.externalLoadBalancer = replica
		}

	}

	// if more than one control plane node exists, fixes names to get a progressive index
	if len(d.controlPlanes) > 1 {
		for i, n := range d.controlPlanes {
			n.Name = fmt.Sprintf("%s%d", "control-plane", i+1)
		}
	}

	// if more than one worker node exists, fixes names to get a progressive index
	if len(d.workers) > 1 {
		for i, n := range d.workers {
			n.Name = fmt.Sprintf("%s%d", "worker", i+1)
		}
	}

	// ensure the list of nodes is ordered.
	// the ordering is key for getting a consistent and predictable behaviour
	// when provisioning nodes and when executing actions on nodes
	sort.Sort(d.allReplicas)

	return nil
}

// AllReplicas returns all the node replicas defined in the `kind` Config.
func (d *DerivedConfig) AllReplicas() ReplicaList {
	return d.allReplicas
}

// ControlPlanes returns all the nodes with control-plane role
func (d *DerivedConfig) ControlPlanes() ReplicaList {
	return d.controlPlanes
}

// BootStrapControlPlane returns the first node with control-plane role
// This is the node where kubeadm init will be executed.
func (d *DerivedConfig) BootStrapControlPlane() *NodeReplica {
	if len(d.controlPlanes) == 0 {
		return nil
	}
	return d.controlPlanes[0]
}

// SecondaryControlPlanes returns all the nodes with control-plane role
// except the BootStrapControlPlane node, if any,
func (d *DerivedConfig) SecondaryControlPlanes() ReplicaList {
	if len(d.controlPlanes) <= 1 {
		return nil
	}
	return d.controlPlanes[1:]
}

// Workers returns all the nodes with Worker role, if any
func (d *DerivedConfig) Workers() ReplicaList {
	return d.workers
}

// ExternalEtcd returns the node with external-etcd role, if defined
func (d *DerivedConfig) ExternalEtcd() *NodeReplica {
	return d.externalEtcd
}

// ExternalLoadBalancer returns the node with external-load-balancer role, if defined
func (d *DerivedConfig) ExternalLoadBalancer() *NodeReplica {
	return d.externalLoadBalancer
}
