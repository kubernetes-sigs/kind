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

package create

import (
	"fmt"
	"os"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/internal/delete"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Options holds cluster creation options
// NOTE: this is only exported for usage by ./../create
type Options struct {
	Retain       bool
	WaitForReady time.Duration
}

// Cluster creates a cluster
func Cluster(c *context.Context, cfg *config.Config, opts *Options) error {
	// default config fields (important for usage as a library, where the config
	// may be constructed in memory rather than from disk)
	encoding.Scheme.Default(cfg)

	// then validate
	if err := cfg.Validate(); err != nil {
		return err
	}

	// derive info necessary for creation
	derived, err := Derive(cfg)
	if err != nil {
		return err
	}

	// init the create context and logging
	// TODO(bentheelder): eliminate this
	cc := &Context{
		Config:        cfg,
		DerivedConfig: derived,
		Context:       c,
	}

	cc.Status = logutil.NewStatus(os.Stdout)
	cc.Status.MaybeWrapLogrus(log.StandardLogger())

	defer cc.Status.End(false)

	// TODO(bentheelder): eliminate create context
	if opts.Retain {
		cc.Retain = true
	}
	if opts.WaitForReady != time.Duration(0) {
		cc.ExecOptions = append(cc.ExecOptions, WaitForReady(opts.WaitForReady))
	}

	// attempt to explicitly pull the required node images if they doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	cc.EnsureNodeImages()

	// Create node containers implementing defined config Nodes
	nodeList, err := cc.ProvisionNodes()
	if err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		log.Error(err)
		if !cc.Retain {
			delete.Cluster(c)
		}
		return err
	}
	cc.Status.End(true)

	// After creating node containers the Kubernetes provisioning is executed
	// By default `kind` executes all the actions required to get a fully working
	// Kubernetes cluster; please note that the list of actions automatically
	// adapt to the topology defined in config
	// TODO(fabrizio pandini): make the list of executed actions configurable from CLI
	err = cc.Exec(nodeList, []string{"haproxy", "config", "init", "join"}, cc.ExecOptions...)
	if err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		log.Error(err)
		if !cc.Retain {
			delete.Cluster(c)
		}
		return err
	}

	// TODO: consider shell detection.
	if runtime.GOOS == "windows" {
		fmt.Printf(
			"Cluster creation complete. To setup KUBECONFIG:\n\n"+

				"For the default cmd.exe console call:\n"+
				"kind get kubeconfig-path > kindpath\n"+
				"set /p KUBECONFIG=<kindpath && del kindpath\n\n"+

				"for PowerShell call:\n"+
				"$env:KUBECONFIG=\"$(kind get kubeconfig-path --name=%[1]q)\"\n\n"+

				"For bash on Windows:\n"+
				"export KUBECONFIG=\"$(kind get kubeconfig-path --name=%[1]q)\"\n\n"+

				"You can now use the cluster:\n"+
				"kubectl cluster-info\n",
			cc.Name(),
		)
	} else {
		fmt.Printf(
			"Cluster creation complete. You can now use the cluster with:\n\n"+

				"export KUBECONFIG=\"$(kind get kubeconfig-path --name=%q)\"\n"+
				"kubectl cluster-info\n",
			cc.Name(),
		)
	}
	return nil
}
