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

// NoopLeveledLogger no-ops everything
// This is the default until an actual logger is installed with SetDefault
type NoopLeveledLogger struct{}

// V implements V from the LeveledLogger interfaces
func (n NoopLeveledLogger) V(Level) Logger {
	return NoopLogger{}
}

// NoopLogger does not actually log but implements the Logger interface
type NoopLogger struct{}

// Info implements a no-op Info from the Logger interface
func (n NoopLogger) Info(args ...interface{}) {}

// Infoln implements a no-op Infoln from the Logger interface
func (n NoopLogger) Infoln(args ...interface{}) {}

// Infof implements a no-op Infof from the Logger interface
func (n NoopLogger) Infof(format string, args ...interface{}) {}
