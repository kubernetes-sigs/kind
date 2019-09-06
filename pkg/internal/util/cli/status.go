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

package cli

import (
	"fmt"
	"io"

	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/util/env"
	"sigs.k8s.io/kind/pkg/internal/util/logger"
)

// Status is used to track ongoing status in a CLI, with a nice loading spinner
// when attached to a terminal
type Status struct {
	spinner *spinner
	status  string
	logger  log.Logger
}

// NewStatus creates a new default Status
func NewStatus(l log.Logger) *Status {
	s := &Status{
		logger: l,
	}
	if v, ok := l.(*logger.Default); ok {
		s.spinner = newSpinner(v.Writer)
		v.Writer = s.maybeWrapWriter(v.Writer)
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
	if _, err := ww.inner.Write([]byte("\r")); err != nil {
		return 0, err
	}
	n, err = ww.inner.Write(p)
	ww.status.spinner.Start()
	return n, err
}

// maybeWrapWriter returns a FriendlyWriter for w IFF w is a terminal, otherwise it returns w
func (s *Status) maybeWrapWriter(w io.Writer) io.Writer {
	if env.IsTerminal(w) {
		return &FriendlyWriter{
			status: s,
			inner:  w,
		}
	}
	return w
}

// Start starts a new phase of the status, if attached to a terminal
// there will be a loading spinner with this status
func (s *Status) Start(status string) {
	s.End(true)
	// set new status
	s.status = status
	if s.spinner != nil {
		s.spinner.SetSuffix(fmt.Sprintf(" %s ", s.status))
		s.spinner.Start()
	} else {
		s.logger.V(0).Infof(" • %s  ...\n", s.status)
	}
}

// End completes the current status, ending any previous spinning and
// marking the status as success or failure
func (s *Status) End(success bool) {
	if s.status == "" {
		return
	}

	if s.spinner != nil {
		s.spinner.Stop()
		fmt.Fprint(s.spinner.writer, "\r")
	}

	if success {
		s.logger.V(0).Infof(" ✓ %s\n", s.status)
	} else {
		s.logger.V(0).Infof("✗ %s\n", s.status)
	}

	s.status = ""
}
