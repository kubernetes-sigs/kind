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

// Package delete contains the logic for performing cluster deletion.
package delete

import (
	"time"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/kubeconfig"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
)

// Cluster deletes the cluster identified by ctx
// explicitKubeconfigPath is --kubeconfig, following the rules from
// https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands
func Cluster(logger log.Logger, p providers.Provider, name, explicitKubeconfigPath string) error {
	n, err := p.ListNodes(name)
	if err != nil {
		return errors.Wrap(err, "error listing nodes")
	}

	kerr := retryKubeconfigUpdate(func() error {
		return kubeconfig.Remove(name, explicitKubeconfigPath)
	})
	if kerr != nil {
		logger.Errorf("failed to update kubeconfig: %v", kerr)
	}

	if len(n) > 0 {
		err = p.DeleteNodes(n)
		if err != nil {
			return err
		}
		logger.V(0).Infof("Deleted nodes: %q", n)
	}

	if kerr != nil {
		return kerr
	}

	return nil
}

// kubeconfigUpdateBackoff matches the backoff used when exporting the
// kubeconfig during cluster creation
var kubeconfigUpdateBackoff = []time.Duration{0, time.Millisecond, time.Millisecond * 50, time.Millisecond * 100}

// retryKubeconfigUpdate calls update with a bounded backoff so that concurrent
// kind invocations contending for the KUBECONFIG lock file do not fail
// immediately
func retryKubeconfigUpdate(update func() error) error {
	var err error
	for _, b := range kubeconfigUpdateBackoff {
		time.Sleep(b)
		if err = update(); err == nil {
			return nil
		}
	}
	return err
}
