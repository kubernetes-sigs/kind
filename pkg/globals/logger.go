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

// package globals will be deleted in a future commit
// this package contains globals that we've not yet re-worked to not be globals
package globals

import (
	"os"
	"sync"

	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/util/logger"
)

// TODO: consider threading through a logger instead of using a global logger
// This version is just a first small step towards making kind easier to import
// in test-harnesses
var globalLoggerMu sync.Mutex
var globalLogger log.Logger = log.NoopLogger{}

// SetLogger sets the standard logger used by this package.
// If not set, log.NoopLogger will be used
func SetLogger(l log.Logger) {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	globalLogger = l
}

// UseDefaultLogger sets the global logger to kind's default logger
// with SetLogger
//
// Not to be confused with the default if not set of log.NoopLogger
func UseDefaultLogger(verbosity log.Level) {
	SetLogger(&logger.Default{
		Verbosity: verbosity,
		Writer:    os.Stderr,
	})
}

// GetLogger returns the standard logger used by this package
func GetLogger() log.Logger {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	return globalLogger
}
