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

// Package status contains functionality for showing CLI application status
package status

import (
	"fmt"
	"io"

	"sigs.k8s.io/kind/pkg/env"
	"sigs.k8s.io/kind/pkg/log/fidget"

	"github.com/sirupsen/logrus"
)

// Status is used to track ongoing status in a CLI, with a nice loading spinner
// when attached to a terminal
type Status struct {
	spinner *fidget.Spinner
	status  string
	writer  io.Writer
}

// New creates a new default Status
func New(w io.Writer) *Status {
	spin := fidget.NewSpinner(w)
	s := &Status{
		spinner: spin,
		writer:  w,
	}
	return s
}

// FriendlyWriter is used to wrap another Writer to make it toggle the
// status spinner before and after writes so that they do not collide
type FriendlyWriter struct {
	status *Status
	inner  io.Writer
}

var _ io.Writer = &FriendlyWriter{}

func (ww *FriendlyWriter) Write(p []byte) (n int, err error) {
	ww.status.spinner.Stop()
	ww.inner.Write([]byte("\r"))
	n, err = ww.inner.Write(p)
	ww.status.spinner.Start()
	return n, err
}

// WrapWriter returns a FriendlyWriter for w
func (s *Status) WrapWriter(w io.Writer) io.Writer {
	return &FriendlyWriter{
		status: s,
		inner:  w,
	}
}

// WrapLogrus wraps a logrus logger's output with a FriendlyWriter
func (s *Status) WrapLogrus(logger *logrus.Logger) {
	logger.SetOutput(s.WrapWriter(logger.Out))
}

// MaybeWrapWriter returns a StatusFriendlyWriter for w IFF w and spinner's
// output are a terminal, otherwise it returns w
func (s *Status) MaybeWrapWriter(w io.Writer) io.Writer {
	if env.IsTerminal(s.writer) && env.IsTerminal(w) {
		return s.WrapWriter(w)
	}
	return w
}

// MaybeWrapLogrus behaves like MaybeWrapWriter for a logrus logger
func (s *Status) MaybeWrapLogrus(logger *logrus.Logger) {
	logger.SetOutput(s.MaybeWrapWriter(logger.Out))
}

// Start starts a new phase of the status, if attached to a terminal
// there will be a loading spinner with this status
func (s *Status) Start(status string) {
	s.End(true)
	// set new status
	isTerm := env.IsTerminal(s.writer)
	s.status = status
	if isTerm {
		s.spinner.SetSuffix(fmt.Sprintf(" %s ", s.status))
		s.spinner.Start()
	} else {
		fmt.Fprintf(s.writer, " • %s  ...\n", s.status)
	}
}

// End completes the current status, ending any previous spinning and
// marking the status as success or failure
func (s *Status) End(success bool) {
	if s.status == "" {
		return
	}

	isTerm := env.IsTerminal(s.writer)
	if isTerm {
		s.spinner.Stop()
		fmt.Fprint(s.writer, "\r")
	}

	if success {
		fmt.Fprintf(s.writer, " ✓ %s\n", s.status)
	} else {
		fmt.Fprintf(s.writer, " ✗ %s\n", s.status)
	}

	s.status = ""
}
