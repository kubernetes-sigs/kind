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

// Package log contains logging related functionality
package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// Status is used to track ongoing status in a CLI, with a nice loading spinner
// when attached to a terminal
type Status struct {
	spinner *spinner.Spinner
	status  string
}

// NewStatus creates a new default Status
func NewStatus() *Status {
	spin := spinner.New(
		spinnerFrames,
		100*time.Millisecond,
	)
	spin.Writer = os.Stdout
	s := &Status{
		spinner: spin,
	}
	return s
}

// StatusFriendlyWriter is used to wrap another Writer to make it toggle the
// status spinner before and after writes so that they do not collide
type StatusFriendlyWriter struct {
	status *Status
	inner  io.Writer
}

var _ io.Writer = &StatusFriendlyWriter{}

func (ww *StatusFriendlyWriter) Write(p []byte) (n int, err error) {
	ww.status.spinner.Stop()
	n, err = ww.inner.Write(p)
	ww.status.spinner.Start()
	return n, err
}

// WrapWriter returns a StatusFriendlyWriter for w
func (s *Status) WrapWriter(w io.Writer) io.Writer {
	return &StatusFriendlyWriter{
		status: s,
		inner:  w,
	}
}

// WrapLogrus wraps a logrus logger's output with a StatusFriendlyWriter
func (s *Status) WrapLogrus(logger *logrus.Logger) {
	logger.SetOutput(s.WrapWriter(logger.Out))
}

// MaybeWrapWriter returns a StatusFriendlyWriter for w IFF w and spinner's
// output are a terminal, otherwise it returns w
func (s *Status) MaybeWrapWriter(w io.Writer) io.Writer {
	if IsTerminal(s.spinner.Writer) && IsTerminal(w) {
		return s.WrapWriter(w)
	}
	return w
}

// MaybeWrapLogrus behaves like MaybeWrapWriter for a logrus logger
func (s *Status) MaybeWrapLogrus(logger *logrus.Logger) {
	logger.SetOutput(s.MaybeWrapWriter(logger.Out))
}

var spinnerFrames = []string{
	"⠈⠁",
	"⠈⠑",
	"⠈⠱",
	"⠈⡱",
	"⢀⡱",
	"⢄⡱",
	"⢄⡱",
	"⢆⡱",
	"⢎⡱",
	"⢎⡰",
	"⢎⡠",
	"⢎⡀",
	"⢎⠁",
	"⠎⠁",
	"⠊⠁",
}

// IsTerminal returns true if the writer w is a terminal
func IsTerminal(w io.Writer) bool {
	if v, ok := (w).(*os.File); ok {
		return terminal.IsTerminal(int(v.Fd()))
	}
	return false
}

// Start starts a new phase of the status, if attached to a terminal
// there will be a loading spinner with this status
func (s *Status) Start(status string) {
	s.End(true)
	// set new status
	isTerm := IsTerminal(s.spinner.Writer)
	s.status = status
	if isTerm {
		s.spinner.Suffix = fmt.Sprintf(" %s ", s.status)
		s.spinner.Start()
	} else {
		fmt.Fprintf(s.spinner.Writer, " • %s  ...\n", s.status)
	}
}

// End completes the current status, ending any previous spinning and
// marking the status as success or failure
func (s *Status) End(success bool) {
	if s.status == "" {
		return
	}

	isTerm := IsTerminal(s.spinner.Writer)
	if isTerm {
		s.spinner.Stop()
	}

	if success {
		fmt.Fprintf(s.spinner.Writer, " ✓ %s\n", s.status)
	} else {
		fmt.Fprintf(s.spinner.Writer, " ✗ %s\n", s.status)
	}

	s.status = ""
}
