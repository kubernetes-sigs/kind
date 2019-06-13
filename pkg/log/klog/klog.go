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

// Package klog contains logging related functionality
package klog

import (
	"k8s.io/klog"

	"sigs.k8s.io/kind/pkg/log"
)

// Use installs this logger as the default
func Use() {
	log.SetDefault(New())
}

// New returns an opaque implementation of log.LeveledLogger against klog
func New() log.LeveledLogger {
	return &leveledLogger{}
}

// leveledLogger implements log.LeveledLogger against klog
type leveledLogger struct{}

var _ log.LeveledLogger = &leveledLogger{}

func (l *leveledLogger) V(level log.Level) log.Logger {
	return klog.V(klog.Level(level))
}
