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

// Level is a leveled logging level, see https://github.com/kubernetes/klog
type Level int32

// LeveledLogger is a subset of what https://github.com/kubernetes/klog
// provides that kind's library code may use. You can (and should) inject your
// own default logger using SetDefault()
//
// See also klog.Use() in our klog package under this one
type LeveledLogger interface {
	V(Level) Logger
}

// Logger is an interface like klog.Verbose
// see: https://github.com/kubernetes/klog
type Logger interface {
	Info(args ...interface{})
	Infoln(args ...interface{})
	Infof(format string, args ...interface{})
}
