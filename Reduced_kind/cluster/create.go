package cluster

import (
	"fmt"

	"reducedkind/config"
)

// Action is one step in the post-provisioning pipeline.  Each step receives
// the node list, the resolved config, and the provider it was launched on.
//
// The pipeline is small on purpose: kind has 7 actions, Reduced_kind has 4.
// The dropped ones (loadbalancer, installstorage, separate config-write step)
// are either multi-CP-only or can be folded into kubeadm config templates.
type Action func(nodes []Node, cfg *config.Cluster, p Provider) error

// Create brings up a full Kubernetes cluster:
//
//  1. provider.Provision     starts every node container
//  2. <pipeline of Actions>  configures Kubernetes inside those containers
//
// The pipeline is supplied by the caller so that the runtime backend
// (single-host or multi-host) can reuse the orchestration unchanged.
func Create(p Provider, cfg *config.Cluster, pipeline []Action) error {
	config.SetDefaultsCluster(cfg)

	// Refuse to clobber an existing cluster.  Caller must `delete` first.
	existing, err := p.ListNodes(cfg.Name)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return fmt.Errorf("cluster %q already exists (%d nodes); run `delete %s` first",
			cfg.Name, len(existing), cfg.Name)
	}

	if err := p.Provision(cfg); err != nil {
		return err
	}

	nodes, err := p.ListNodes(cfg.Name)
	if err != nil {
		return err
	}

	for _, step := range pipeline {
		if err := step(nodes, cfg, p); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes every container associated with clusterName.
func Delete(p Provider, clusterName string) error {
	nodes, err := p.ListNodes(clusterName)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}
	return p.DeleteNodes(nodes)
}

// FindByRole returns the first node matching role, or nil.
func FindByRole(nodes []Node, role string) Node {
	for _, n := range nodes {
		if n.Role() == role {
			return n
		}
	}
	return nil
}

// FilterByRole returns every node matching role.
func FilterByRole(nodes []Node, role string) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Role() == role {
			out = append(out, n)
		}
	}
	return out
}
