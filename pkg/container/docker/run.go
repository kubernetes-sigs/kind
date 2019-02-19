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

package docker

import (
	"regexp"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/container/cri"
	"sigs.k8s.io/kind/pkg/exec"
)

// Docker container IDs are hex, more than one character, and on their own line
var containerIDRegex = regexp.MustCompile("^[a-f0-9]+$")

// RunOpt is an option for Run
type RunOpt func(*runOpts) *runOpts

// actual options struct
// TODO(bentheelder): these options are likely not the right or final
type runOpts struct {
	RunArgs       []string
	ContainerArgs []string
	Mounts        []cri.Mount
}

// WithRunArgs sets the args for docker run
// as in the args portion of `docker run args... image containerArgs...`
func WithRunArgs(args ...string) RunOpt {
	return func(r *runOpts) *runOpts {
		r.RunArgs = args
		return r
	}
}

// WithContainerArgs sets the args to the container
// as in the containerArgs portion of `docker run args... image containerArgs...`
// NOTE: this is only the args portion before the image
func WithContainerArgs(args ...string) RunOpt {
	return func(r *runOpts) *runOpts {
		r.ContainerArgs = args
		return r
	}
}

// WithMounts sets the container mounts
func WithMounts(mounts []cri.Mount) RunOpt {
	return func(r *runOpts) *runOpts {
		r.Mounts = mounts
		return r
	}
}

// Run creates a container with "docker run", with some error handling
// it will return the ID of the created container if any, even on error
func Run(image string, opts ...RunOpt) (id string, err error) {
	o := &runOpts{}
	for _, opt := range opts {
		o = opt(o)
	}
	// convert mounts to container run args
	runArgs := o.RunArgs
	for _, mount := range o.Mounts {
		runArgs = append(runArgs, generateMountBindings(mount)...)
	}
	// construct the actual docker run argv
	args := []string{"run"}
	args = append(args, runArgs...)
	args = append(args, image)
	args = append(args, o.ContainerArgs...)
	cmd := exec.Command("docker", args...)
	output, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		// log error output if there was any
		for _, line := range output {
			log.Error(line)
		}
		return "", err
	}
	// if docker created a container the id will be the first line and match
	// validate the output and get the id
	if len(output) < 1 {
		return "", errors.New("failed to get container id, received no output from docker run")
	}
	if !containerIDRegex.MatchString(output[0]) {
		return "", errors.Errorf("failed to get container id, output did not match: %v", output)
	}
	return output[0], nil
}
