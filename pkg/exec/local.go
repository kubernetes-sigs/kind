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
	"fmt"
	"io"
	osexec "os/exec"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/pkg/errors"

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
func (cmd *LocalCmd) Run() error {
	err := &localCmdError{
		CmdArgs: cmd.Args,
	}
	// TODO(BenTheElder): adding bytes.Buffer to both multiwriters might need
	// to be wrapped with a mutex
	if cmd.Stdout != nil {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, &err.CmdOut)
	} else {
		cmd.Stdout = &err.CmdOut
	}
	if cmd.Stderr != nil {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, &err.CmdOut)
	} else {
		cmd.Stderr = &err.CmdOut
	}
	// TODO: should be in the caller or logger should be injected somehow ...
	globals.GetLogger().V(3).Infof("Running: \"%s\"", err.prettyCommand())
	if e := cmd.Cmd.Run(); e != nil {
		err.Inner = e
		return errors.WithStack(err)
	}
	return nil
}

type localCmdError struct {
	CmdArgs []string
	CmdOut  bytes.Buffer
	Inner   error
}

// TODO(BenTheElder): implement formatter instead, and only show
// output for %+v ?
func (e *localCmdError) Error() string {
	return fmt.Sprintf(
		"command \"%s\" failed with error: %v and output:\n%s",
		e.prettyCommand(), e.Inner, e.CmdOut.Bytes(),
	)
}

func (e *localCmdError) prettyCommand() string {
	var out strings.Builder
	for i, arg := range e.CmdArgs {
		out.WriteString(shellescape.Quote(arg))
		if i+1 != len(e.CmdArgs) {
			out.WriteByte(' ')
		}
	}
	return out.String()
}

// Cause mimics github.com/pkg/errors's Cause pattern for errors
func (e *localCmdError) Cause() error {
	return e.Inner
}
