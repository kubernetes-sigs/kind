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

package exec

import (
	"bytes"
	"io"
	osexec "os/exec"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/globals"
)

// LocalCmd wraps os/exec.Cmd, implementing the kind/pkg/exec.Cmd interface
type LocalCmd struct {
	*osexec.Cmd
}

var _ Cmd = &LocalCmd{}

// LocalCmder is a factory for LocalCmd, implementing Cmder
type LocalCmder struct{}

var _ Cmder = &LocalCmder{}

// Command returns a new exec.Cmd backed by Cmd
func (c *LocalCmder) Command(name string, arg ...string) Cmd {
	return &LocalCmd{
		Cmd: osexec.Command(name, arg...),
	}
}

// SetEnv sets env
func (cmd *LocalCmd) SetEnv(env ...string) Cmd {
	cmd.Env = env
	return cmd
}

// SetStdin sets stdin
func (cmd *LocalCmd) SetStdin(r io.Reader) Cmd {
	cmd.Stdin = r
	return cmd
}

// SetStdout set stdout
func (cmd *LocalCmd) SetStdout(w io.Writer) Cmd {
	cmd.Stdout = w
	return cmd
}

// SetStderr sets stderr
func (cmd *LocalCmd) SetStderr(w io.Writer) Cmd {
	cmd.Stderr = w
	return cmd
}

// Run runs the command
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Run() error {
	var out bytes.Buffer
	// TODO(BenTheElder): adding bytes.Buffer to both multiwriters might need
	// to be wrapped with a mutex
	if cmd.Stdout != nil {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, &out)
	} else {
		cmd.Stdout = &out
	}
	if cmd.Stderr != nil {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, &out)
	} else {
		cmd.Stderr = &out
	}
	// TODO: should be in the caller or logger should be injected somehow ...
	globals.GetLogger().V(3).Infof("Running: \"%s\"", PrettyCommand(cmd.Args[0], cmd.Args[1:]...))
	if err := cmd.Cmd.Run(); err != nil {
		return errors.WithStack(&RunError{
			Command: cmd.Args,
			Output:  out.Bytes(),
			Inner:   err,
		})
	}
	return nil
}
