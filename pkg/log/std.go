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

package log

import (
	"sync"
)

// TODO: consider threading through a logger instead of using a global logger
// This version is just a first small step towards making kind easier to import
// in test-harnesses
var stdmu sync.Mutex
var std Logger = NoopLogger{}

// Set sets the standard logger used by this package
func Set(logger Logger) {
	stdmu.Lock()
	defer stdmu.Unlock()
	std = logger
}

// Get returns the standard logger used by this package
func Get() Logger {
	stdmu.Lock()
	defer stdmu.Unlock()
	return std
}

// Warn wraps Warn on the standard logger
func Warn(message string) {
	Get().Warn(message)
}

// Warnf wraps Warnf on the standard logger
func Warnf(format string, args ...interface{}) {
	Get().Warnf(format, args...)
}

// Error wraps Error on the standard logger
func Error(message string) {
	Get().Error(message)
}

// Errorf wraps Errorf on the standard logger
func Errorf(format string, args ...interface{}) {
	Get().Errorf(format, args...)
}

// V wraps V on the standard logger
func V(level Level) InfoLogger {
	return Get().V(level)
}
