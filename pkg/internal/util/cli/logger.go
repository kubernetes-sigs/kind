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

package cli

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"sigs.k8s.io/kind/pkg/log"
)

// Logger is the kind cli's log.Logger implementation
type Logger struct {
	Verbosity log.Level
	io.Writer
	writeMu sync.Mutex
}

var _ log.Logger = &Logger{}

// NewLogger returns a new Logger with the given verbosity
func NewLogger(writer io.Writer, verbosity log.Level) *Logger {
	return &Logger{
		Verbosity: verbosity,
		Writer:    writer,
	}
}

func (l *Logger) Write(p []byte) (n int, err error) {
	// TODO: line oriented instead?
	// For now we make a single per-message write call from the rest of the logger
	// intentionally to effectively do this one level up
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	return l.Writer.Write(p)
}

// TODO: prefix log lines with metadata (log level? timestamp?)

func (l *Logger) print(message string) {
	buf := bytes.NewBufferString(message)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	// TODO: should we handle this somehow??
	// Who logs for the logger? 🤔
	_, _ = l.Write(buf.Bytes())
}

func (l *Logger) printf(format string, args ...interface{}) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, format, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	// TODO: should we handle this somehow??
	// Who logs for the logger? 🤔
	_, _ = l.Write(buf.Bytes())
}

// Warn is part of the log.Logger interface
func (l *Logger) Warn(message string) {
	l.print(message)
}

// Warnf is part of the log.Logger interface
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.printf(format, args...)
}

// Error is part of the log.Logger interface
func (l *Logger) Error(message string) {
	l.print(message)
}

// Errorf is part of the log.Logger interface
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.printf(format, args...)
}

// V is part of the log.Logger interface
func (l *Logger) V(level log.Level) log.InfoLogger {
	return infoLogger{
		logger:  l,
		enabled: level <= l.Verbosity,
	}
}

type infoLogger struct {
	logger  *Logger
	enabled bool
}

func (i infoLogger) Enabled() bool {
	return i.enabled
}

func (i infoLogger) Info(message string) {
	if !i.enabled {
		return
	}
	i.logger.print(message)
}

func (i infoLogger) Infof(format string, args ...interface{}) {
	if !i.enabled {
		return
	}
	i.logger.printf(format, args...)
}
