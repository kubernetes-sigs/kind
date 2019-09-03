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
	"sync"
	"time"
)

// custom CLI loading spinner for kind
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

// spinner is a simple and efficient CLI loading spinner used by kind
// It is simplistic and assumes that the line length will not change.
// It is best used indirectly via log.Status (see parent package)
type spinner struct {
	frames []string
	stop   chan struct{}
	ticker *time.Ticker
	writer io.Writer
	mu     *sync.Mutex
	// protected by mu
	prefix string
	suffix string
}

// Newspinner initializes and returns a new spinner that will write to
func newSpinner(w io.Writer) *spinner {
	return &spinner{
		frames: spinnerFrames,
		stop:   make(chan struct{}, 1),
		ticker: time.NewTicker(time.Millisecond * 100),
		mu:     &sync.Mutex{},
		writer: w,
	}
}

// SetPrefix sets the prefix to print before the spinner
func (s *spinner) SetPrefix(prefix string) {
	s.mu.Lock()
	s.prefix = prefix
	s.mu.Unlock()
}

// SetSuffix sets the suffix to print after the spinner
func (s *spinner) SetSuffix(suffix string) {
	s.mu.Lock()
	s.suffix = suffix
	s.mu.Unlock()
}

// Start starts the spinner running
func (s *spinner) Start() {
	go func() {
		for {
			for _, frame := range s.frames {
				select {
				case <-s.stop:
					return
				case <-s.ticker.C:
					func() {
						s.mu.Lock()
						defer s.mu.Unlock()
						fmt.Fprintf(s.writer, "\r%s%s%s", s.prefix, frame, s.suffix)
					}()
				}
			}
		}
	}()
}

// Stop signals the spinner to stop
func (s *spinner) Stop() {
	s.stop <- struct{}{}
}
