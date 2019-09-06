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

package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/util/env"
)

// Default is the default log.Logger implementation
type Default struct {
	Verbosity log.Level
	io.Writer
	writeMu sync.Mutex
	// for later use in adding colored output etc...
	// we collect this in NewDefault before any writer wrapping may occur
	isTerm bool
}

var _ log.Logger = &Default{}

// NewDefault returns a new Default logger with the given verbosity
func NewDefault(verbosity log.Level) *Default {
	return &Default{
		Verbosity: verbosity,
		Writer:    os.Stderr,
		isTerm:    env.IsTerminal(os.Stderr),
	}
}

func (d *Default) Write(p []byte) (n int, err error) {
	// TODO: line oriented instead?
	// For now we make a single per-message write call from the rest of the logger
	// intentionally to effectively do this one level up
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return d.Writer.Write(p)
}

// TODO: prefix log lines with metadata (log level? timestamp?)

func (d *Default) print(message string) {
	buf := bytes.NewBufferString(message)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	d.Write(buf.Bytes())
}

func (d *Default) printf(format string, args ...interface{}) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, format, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	d.Write(buf.Bytes())
}

// Warn is part of the log.Logger interface
func (d *Default) Warn(message string) {
	d.print(message)
}

// Warnf is part of the log.Logger interface
func (d *Default) Warnf(format string, args ...interface{}) {
	d.printf(format, args...)
}

// Error is part of the log.Logger interface
func (d *Default) Error(message string) {
	d.print(message)
}

// Errorf is part of the log.Logger interface
func (d *Default) Errorf(format string, args ...interface{}) {
	d.printf(format, args...)
}

// V is part of the log.Logger interface
func (d *Default) V(level log.Level) log.InfoLogger {
	return defaultInfo{
		logger:  d,
		enabled: level <= d.Verbosity,
	}
}

type defaultInfo struct {
	logger  *Default
	enabled bool
}

func (d defaultInfo) Enabled() bool {
	return d.enabled
}

func (d defaultInfo) Info(message string) {
	if !d.enabled {
		return
	}
	d.logger.print(message)
}

func (d defaultInfo) Infof(format string, args ...interface{}) {
	if !d.enabled {
		return
	}
	d.logger.printf(format, args...)
}
