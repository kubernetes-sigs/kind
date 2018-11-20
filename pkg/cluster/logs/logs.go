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

package logs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/util"
)

type errFn func() error

// Collect collects logs related to / from the cluster nodes and the host
// system to the specified directory
func Collect(nodes []nodes.Node, dir string) error {
	prefixedPath := func(path string) string {
		return filepath.Join(dir, path)
	}
	// helper to run a cmd and write the output to path
	execToPath := func(cmd exec.Cmd, path string) error {
		realPath := prefixedPath(path)
		os.MkdirAll(filepath.Dir(realPath), os.ModePerm)
		f, err := os.Create(realPath)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd.SetStdout(f)
		cmd.SetStderr(f)
		return cmd.Run()
	}
	execToPathFn := func(cmd exec.Cmd, path string) func() error {
		return func() error {
			return execToPath(cmd, path)
		}
	}
	// construct a slice of methods to collect logs
	fns := []errFn{
		// TODO(bentheelder): record the kind version here as well
		// record info about the host docker
		execToPathFn(
			exec.Command("docker", "info"),
			"docker-info.txt",
		),
	}
	// add a log collection method for each node
	for _, n := range nodes {
		node := n // https://golang.org/doc/faq#closures_and_goroutines
		fns = append(fns, func() error {
			name := node.String()
			return coalesce(
				// record info about the node container
				execToPathFn(
					exec.Command("docker", "inspect", name),
					filepath.Join(name, "inspect.json"),
				),
				// grab all of the node logs
				execToPathFn(
					node.Command("cat", "/kind/version"),
					filepath.Join(name, "kubernetes-version.txt"),
				),
				execToPathFn(
					node.Command("journalctl", "--no-pager"),
					filepath.Join(name, "journal.log"),
				),
				execToPathFn(
					node.Command("journalctl", "--no-pager", "-u", "kubelet.service"),
					filepath.Join(name, "kubelet.log"),
				),
				execToPathFn(
					node.Command("journalctl", "--no-pager", "-u", "docker.service"),
					filepath.Join(name, "docker.log"),
				),
				// grab all container / pod logs
				// NOTE: we cannot just docker cp this directory because
				// it is mostly symlinks, so we must list and resolve the symlinks
				func() error {
					// collect up file names
					files, err := exec.CombinedOutputLines(
						node.Command("find", "-L", "/var/log", "-type", "f"),
					)
					if err != nil {
						return err
					}
					// for each file, we want to copy it out at the path
					// we find it (symlink or not), so we pipe cat in the container
					// to a file in our logs dir on the host
					copyFns := []errFn{}
					for _, f := range files {
						file := f // https://golang.org/doc/faq#closures_and_goroutines
						targetPath := strings.TrimPrefix(file, "/var/log/")
						copyFns = append(copyFns, func() error {
							return execToPath(
								node.Command("cat", file),
								filepath.Join(name, targetPath),
							)
						})
					}
					return coalesce(copyFns...)
				},
			)
		})
	}
	// run and collect up all errors
	return coalesce(fns...)
}

// colaese runs fns concurrently, returning an Errors if there are > 1 errors
func coalesce(fns ...errFn) error {
	// run all fns concurrently
	ch := make(chan error, len(fns))
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func(f errFn) {
			defer wg.Done()
			ch <- f()
		}(fn)
	}
	wg.Wait()
	close(ch)
	// collect up and return errors
	errs := []error{}
	for err := range ch {
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 1 {
		return util.Flatten(errs)
	} else if len(errs) == 1 {
		return errs[0]
	}
	return nil
}
