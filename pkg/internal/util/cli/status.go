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

// StatusForLogger returns a new status object for the logger l,
// if l is the kind default logger it will inject a wrapped writer into it
// and if we've already attached one it will return the previous status
func StatusForLogger(l log.Logger) *Status {
	s := &Status{
		logger: l,
	}
	if v, ok := l.(*logger.Default); ok {
		// Be re-entrant and only attach one spinner
		// TODO: how do we handle concurrent spinner instances !?
		// IE: library usage + the default logger ...
		if v2, ok := v.Writer.(*FriendlyWriter); ok {
			return v2.status
		}
		// otherwise wrap the logger's writer for the first time
		if env.IsTerminal(v.Writer) {
			s.spinner = newSpinner(v.Writer)
			v.Writer = &FriendlyWriter{
				status: s,
				inner:  v.Writer,
			}
		}
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
