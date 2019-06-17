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

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/internal/delete"
	logutil "sigs.k8s.io/kind/pkg/log"

	configaction "sigs.k8s.io/kind/pkg/cluster/internal/create/actions/config"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/installcni"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/installstorage"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/kubeadminit"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/kubeadmjoin"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/waitforready"
)

const (
	// Typical host name max limit is 64 characters (https://linux.die.net/man/2/sethostname)
	// We append -control-plane (14 characters) to the cluster name on the control plane container
	clusterNameMax = 50
)

// Options holds cluster creation options
// NOTE: this is only exported for usage by ./../create
type Options struct {
	Retain       bool
	WaitForReady time.Duration
	//TODO: Refactor this. It is a temporary solution for a phased breakdown of different
	//      operations, specifically create. see https://github.com/kubernetes-sigs/kind/issues/324
	SetupKubernetes bool // if kind should setup kubernetes after creating nodes
}

// Cluster creates a cluster
func Cluster(ctx *context.Context, cfg *config.Cluster, opts *Options) error {
	// default config fields (important for usage as a library, where the config
	// may be constructed in memory rather than from disk)
	encoding.Scheme.Default(cfg)

	// warn if cluster name might typically be too long
	if len(ctx.Name()) > clusterNameMax {
		log.Warnf("cluster name %q is probably too long, this might not work properly on some systems", ctx.Name())
	}

	// then validate
	if err := cfg.Validate(); err != nil {
		return err
	}

	status := logutil.NewStatus(os.Stdout)
	status.MaybeWrapLogrus(log.StandardLogger())

	// attempt to explicitly pull the required node images if they doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	ensureNodeImages(status, cfg)

	// Create node containers implementing defined config Nodes
	if err := provisionNodes(status, cfg, ctx.Name(), ctx.ClusterLabel()); err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		log.Error(err)
		if !opts.Retain {
			delete.Cluster(ctx)
		}
		return err
	}

	// TODO(bentheelder): make this controllable from the command line?
	actionsToRun := []actions.Action{
		loadbalancer.NewAction(), // setup external loadbalancer
		configaction.NewAction(), // setup kubeadm config
	}
	if opts.SetupKubernetes {
		actionsToRun = append(actionsToRun,
			kubeadminit.NewAction(), // run kubeadm init
		)
		// this step might be skipped, but is next after init
		if !cfg.Networking.DisableDefaultCNI {
			actionsToRun = append(actionsToRun,
				installcni.NewAction(), // install CNI
			)
		}
		// add remaining steps
		actionsToRun = append(actionsToRun,
			installstorage.NewAction(),                // install StorageClass
			kubeadmjoin.NewAction(),                   // run kubeadm join
			waitforready.NewAction(opts.WaitForReady), // wait for cluster readiness
		)
	}

	// run all actions
	actionsContext := actions.NewActionContext(cfg, ctx, status)
	for _, action := range actionsToRun {
		if err := action.Execute(actionsContext); err != nil {
			if !opts.Retain {
				delete.Cluster(ctx)
			}
			return err
		}
	}

	if !opts.SetupKubernetes {
		// prints how to manually setup the cluster
		printSetupInstruction(ctx.Name())
		return nil
	}

	// print how to set KUBECONFIG to point to the cluster etc.
	printUsage(ctx.Name())

	return nil
}

func printUsage(name string) {
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
			name,
		)
	} else {
		fmt.Printf(
			"Cluster creation complete. You can now use the cluster with:\n\n"+

				"export KUBECONFIG=\"$(kind get kubeconfig-path --name=%q)\"\n"+
				"kubectl cluster-info\n",
			name,
		)
	}
}

func printSetupInstruction(name string) {
	fmt.Printf(
		"Nodes creation complete. You can now setup kubernetes using docker exec %s-<node> kubeadm ...\n",
		name,
	)
}
