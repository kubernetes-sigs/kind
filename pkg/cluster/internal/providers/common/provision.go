/*
Copyright 2021 The Kubernetes Authors.

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

package common

import (
	"bufio"
	"context"
	"fmt"
	"io"
	osexec "os/exec"
	"regexp"
	"time"

	"sigs.k8s.io/kind/pkg/exec"
)

type runContainerOpts struct {
	waitUntilLogRegexpMatches *regexp.Regexp
}

type RunContainerOpt func(*runContainerOpts) error

// WithWaitUntilLogRegexpMatches waits until the container log matches the regexp.
func WithWaitUntilLogRegexpMatches(re *regexp.Regexp) RunContainerOpt {
	return func(o *runContainerOpts) error {
		o.waitUntilLogRegexpMatches = re
		return nil
	}
}

// WithWaitUntilSystemdReachesMultiUserSystem waits until the systemd in the container
// reaches "Multi-User System" target if is using cgroups v2, so that `docker exec` can be
// executed safely without breaking cgroup v2 hierarchy.
//
// This is implemented by grepping `docker logs` with "Reached target .*Multi-User System.*"
// message from systemd.
// (we can't use `docker exec` to check whether we are allowed to run `docker exec`, obviously.)
//
// Needed for avoiding "ERROR: this script needs /sys/fs/cgroup/cgroup.procs to be empty (for writing the top-level cgroup.subtree_control)"
// See https://github.com/kubernetes-sigs/kind/issues/2409
func WithWaitUntilSystemdReachesMultiUserSystem() RunContainerOpt {
	re, err := regexp.Compile("Reached target .*Multi-User System.*|detected cgroup v1")
	if err != nil {
		panic(err)
	}
	return WithWaitUntilLogRegexpMatches(re)
}

// RunContainer runs a container.
// engine is either "docker" or "podman".
// name is the name of the container.
// args is to be appended to {engine, "run", "--name", name}.
func RunContainer(engine, name string, args []string, opts ...RunContainerOpt) error {
	var o runContainerOpts
	for _, f := range opts {
		if err := f(&o); err != nil {
			return err
		}
	}
	fullArgs := append([]string{"run", "--name", name}, args...)
	if err := exec.Command(engine, fullArgs...).Run(); err != nil {
		return fmt.Errorf("%s run error: %w", engine, err)
	}

	if o.waitUntilLogRegexpMatches != nil {
		logCtx := context.Background()
		logCtx, logCancel := context.WithTimeout(logCtx, 30*time.Second)
		defer logCancel()
		// use os/exec.CommandContext directly, as kind/pkg/exec.CommandContext lacks support for killing
		logCmd := osexec.CommandContext(logCtx, engine, "logs", "-f", name)
		pr, pw := io.Pipe()
		defer pr.Close()
		defer pw.Close()
		logCmd.Stdout = pw
		logCmd.Stderr = pw
		if err := logCmd.Start(); err != nil {
			return fmt.Errorf("failed to run %v: %w", logCmd.Args, err)
		}
		defer func() { _ = logCmd.Process.Kill() }()
		return waitUntilLogRegexpMatches(logCtx, pr, o.waitUntilLogRegexpMatches)
	}
	return nil
}

func waitUntilLogRegexpMatches(ctx context.Context, r io.Reader, re *regexp.Regexp) error {
	ch := make(chan string)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			line := sc.Text()
			ch <- line
		}
		close(ch)
	}()
	var errNoMatch = fmt.Errorf("could not find a line that matches %q", re.String())
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return errNoMatch
			}
			if re.MatchString(line) {
				return nil
			}
		case <-ctx.Done():
			return errNoMatch
		}
	}
}
