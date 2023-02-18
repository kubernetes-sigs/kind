package main

import (
	"context"
	"fmt"

	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

var _ cloudprovider.Clusters = &cloud{}

// ListClusters lists the names of the available clusters.
func (c *cloud) ListClusters(ctx context.Context) ([]string, error) {
	klog.V(2).Infof("List clusters")
	return c.kindClient.List()
}

// Master gets back the address (either DNS name or IP address) of the master node for the cluster.
func (c *cloud) Master(ctx context.Context, clusterName string) (string, error) {
	klog.V(2).Infof("Get master for %s", clusterName)
	clusters, err := c.kindClient.List()
	if err != nil {
		return "", err
	}
	for _, cluster := range clusters {
		if cluster == clusterName {
			return clusterName + "-control-plane", nil
		}
	}
	return "", fmt.Errorf("cluster %s node found", clusterName)
}
