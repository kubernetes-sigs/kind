package main

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

var _ cloudprovider.InstancesV2 = (*cloud)(nil)

var errNodeNotFound = errors.New("node not found")

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
func (c *cloud) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	klog.V(2).Infof("Check if instace %s exists", node.Name)
	_, err := c.findNodeByName(node.Name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errNodeNotFound) {
		return false, nil
	}
	return false, err
}

// InstanceShutdown returns true of the container doesn't exist
func (c *cloud) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	klog.V(2).Infof("Check if instace %s is shutdown", node.Name)
	_, err := c.findNodeByName(node.Name)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, errNodeNotFound) {
		return true, nil
	}
	return false, err
}

// InstanceMetadata returns the instance's metadata. The values returned in InstanceMetadata are
// translated into specific fields and labels in the Node object on registration.
func (c *cloud) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.V(2).Infof("Check instace metadata for %s", node.Name)
	n, err := c.findNodeByName(node.Name)
	if err != nil {
		return nil, err
	}
	m := &cloudprovider.InstanceMetadata{
		// TODO: podman support
		ProviderID:   fmt.Sprintf("kind://%s/kind/%s", kindCluster, n.String()), // providerID: kind://<cluster-name>/kind/<node-name>
		InstanceType: "kind-node",
		NodeAddresses: []v1.NodeAddress{
			{
				Type:    v1.NodeHostName,
				Address: n.String(),
			},
		},
		Zone:   "",
		Region: "",
	}
	ipv4, ipv6, err := n.IP()
	if err != nil {
		return nil, err
	}
	if ipv4 != "" {
		m.NodeAddresses = append(m.NodeAddresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: ipv4})
	}
	if ipv6 != "" {
		m.NodeAddresses = append(m.NodeAddresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: ipv6})
	}
	klog.V(2).Infof("instance metadata for %s: %#v", node.Name, m)
	return m, nil
}

func (c *cloud) findNodeByName(name string) (nodes.Node, error) {
	nodes, err := c.kindClient.ListNodes(kindCluster)
	if err != nil {
		return nil, fmt.Errorf("no nodes founds")
	}
	for _, n := range nodes {
		if n.String() == name {
			return n, nil
		}
	}
	return nil, fmt.Errorf("node with name %s does not exist on cluster %s", name, kindCluster)
}
