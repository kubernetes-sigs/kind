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

// Package installcustom implements the an action to install custom
// manifests
package installcustom

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
)

type action struct{}

// NewAction returns a new action for installing custom manifests
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	// skip if there are no custom manifests
	if len(ctx.Config.CustomManifests) == 0 {
		return nil
	}

	ctx.Status.Start("Installing custom manifests 📃")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0] // kind expects at least one always

	// add the custom manifests
	if err := addCustomManifests(node, &ctx.Config.CustomManifests); err != nil {
		return errors.Wrap(err, "failed to deploy manifest")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// run kubectl apply on control plane node, and can be overridden for testing
var runApplyCustomManifest = func(controlPlane nodes.Node, path string, stdin string) error {
	var in *strings.Reader = nil
	// only create if we have stdin
	if len(stdin) > 0 {
		in = strings.NewReader(stdin)
		path = "-"
	}

	cmd := controlPlane.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf", "apply", "-f", path,
	)

	// only close if we had stdin
	if in != nil {
		cmd.SetStdin(in)
	}

	return cmd.Run()
}

func addCustomManifests(controlPlane nodes.Node, customManifests *[]interface{}) (err error) {
	for index, customManifest := range *customManifests {
		var manifestList map[string]string

		// perform conversion to map[string]string
		switch t := (customManifest).(type) {
		// handle file or URL
		case string:
			if strings.HasPrefix(t, "http") {
				// URL is a special case - set contents to empty
				manifestList = map[string]string{t: ""}
			} else {
				// read file in
				var manifest []byte
				if manifest, err = os.ReadFile(t); os.IsNotExist(err) {
					err = fmt.Errorf("customManifests[%d]: '%s' does not exist", index, t)
					return
				}
				manifestList = map[string]string{t: string(manifest)}
			}
		// convert map[string]interface{} to map[string]string
		case map[string]interface{}:
			manifestList = make(map[string]string)
			for manifestName, manifestContents := range t {
				switch manifestContentsString := (manifestContents).(type) {
				case string:
					manifestList[manifestName] = manifestContentsString
				default:
					err = fmt.Errorf("customManifests[%s]: incorrect type (map[string]%T) expected string or map[string]string", manifestName, manifestContentsString)
					return
				}
			}
		case map[string]string:
			manifestList = t
		default:
			err = fmt.Errorf("customManifests[%d]: incorrect type (%T) expected string or map[string]string", index, t)
			return
		}

		// apply all manifest in current array member
		if err == nil {
			for manifestName, manifest := range manifestList {
				path := "-"
				// handle special cases (URL) where content is empty
				if len(manifest) == 0 {
					path = manifestName
				}
				err = runApplyCustomManifest(controlPlane, path, manifest)
				if err != nil {
					err = errors.Wrapf(err, "customManifest[%d][%s]: error deploying manifest", index, manifestName)
					return
				}
			}
		}
	}

	return
}
