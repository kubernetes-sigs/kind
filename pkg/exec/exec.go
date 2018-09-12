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

// Package exec contains helpers for os/exec
package exec

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// Cmd wraps os/exec.Cmd, with extra options for logging etc. when running
type Cmd struct {
	*exec.Cmd
	CmdOpts
}

// CmdOpts contains extra options recognized by this package's os/exec.Cmd
// wrapper (Cmd)
type CmdOpts struct {
	// If Debug is true, log the command prior to running
	Debug bool
	// If LogOutputOnFail is true, stdout and stderr will be collected, and will
	// be logged if the command does not exit cleanly
	LogOutputOnFail bool
	// If InheritOtuput is true, inheret stdout and stderr from the current process
	InheritOutput bool
}

// Command returns a Cmd from os/exec's Command(...)
func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		Cmd: exec.Command(name, arg...),
	}
}

// Run wraps cmd.Run(), respecting CmdOpts
func (cmd *Cmd) Run() error {
	if cmd.Debug {
		log.Infof("Running: %v %v", cmd.Path, cmd.Args)
	}

	if cmd.LogOutputOnFail {
		return cmd.runLoggingOutputOnFail()
	}

	if cmd.InheritOutput {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	}
	return cmd.Cmd.Run()
}

func (cmd *Cmd) runLoggingOutputOnFail() error {
	var buff bytes.Buffer
	cmd.Stdout = &buff
	cmd.Stderr = &buff
	err := cmd.Cmd.Run()
	if cmd.LogOutputOnFail {
		log.Error("failed with:")
		scanner := bufio.NewScanner(&buff)
		for scanner.Scan() {
			log.Error(scanner.Text())
		}
	}
	return err
}

// CombinedOutputLines is like os/exec's cmd.CombinedOutput(),
// except instead of returning the byte buffer of stderr + stdout,
// it scans these for lines and returns a slice of output line strings
func (cmd *Cmd) CombinedOutputLines() (lines []string, err error) {
	if cmd.Debug {
		log.Infof("Running: %v %v", cmd.Path, cmd.Args)
	}

	var buff bytes.Buffer
	cmd.Stdout = &buff
	cmd.Stderr = &buff
	err = cmd.Cmd.Run()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}
