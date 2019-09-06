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

// Level is a verbosity logging level for Info logs
// See also https://github.com/kubernetes/klog
type Level int32

// Logger defines the logging interface kind uses
// It is roughly a subset of github.com/kubernetes/klog
type Logger interface {
	Warn(message string)
	Warnf(format string, args ...interface{})
	Error(message string)
	Errorf(format string, args ...interface{})
	V(Level) InfoLogger
}

// InfoLogger defines the info logging interface kind uses
// It is roughly a subset of Verbose from github.com/kubernetes/klog
type InfoLogger interface {
	Info(message string)
	Infof(format string, args ...interface{})
	Enabled() bool
}
