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

import "io"

// Logger is a interface that should be used for all logging inside kind/pkg.
// Consumers of kind/pkg such as kind/cmd should define an object that satisfies this
// interface and then call SetLogger() passing this object.
// Optionally Output() and SetOutput() can be used if such a logger writes to a specific io.Writer.
// TODO(neolit123): investigate klog/logr integration.
type Logger interface {
	Output() io.Writer
	SetOutput(out io.Writer)

	Error(...interface{})
	Errorf(string, ...interface{})

	Warning(...interface{})
	Warningf(string, ...interface{})

	Info(...interface{})
	Infof(string, ...interface{})

	Debug(...interface{})
	Debugf(string, ...interface{})
}

var logger Logger

// GetLogger returns the locally defined logger that was previously set using SetLogger().
func GetLogger() Logger {
	return logger
}

// SetLogger should be called from another package that consumes kind/pkg.
// The passed logger must satisfy the Logger interface and this logger will be used in all
// logging calls inside kind/pkg.
func SetLogger(l Logger) {
	logger = l
}

// Output returns the defined io.Writer where the kind/pkg logger writes everything.
func Output() io.Writer {
	return logger.Output()
}

// SetOutput sets where the kind/pkg logger writes everything.
func SetOutput(out io.Writer) {
	logger.SetOutput(out)
}

// Error is a local export for the Logger interface.
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf is a local export for the Logger interface.
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Warning is a local export for the Logger interface.
func Warning(args ...interface{}) {
	logger.Warning(args...)
}

// Warningf is a local export for the Logger interface.
func Warningf(format string, args ...interface{}) {
	logger.Warningf(format, args...)
}

// Info is a local export for the Logger interface.
func Info(args ...interface{}) {
	logger.Info(args)
}

// Infof is a local export for the Logger interface.
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Debug is a local export for the Logger interface.
func Debug(args ...interface{}) {
	logger.Debug(args)
}

// Debugf is a local export for the Logger interface.
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}
