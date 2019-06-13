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

var std LeveledLogger = NoopLeveledLogger{}
var stdMu sync.Mutex

// SetDefault sets the default logger
func SetDefault(logger LeveledLogger) {
	stdMu.Lock()
	defer stdMu.Unlock()
	std = logger
}

// V is leveled logging implemented against the default logger
func V(level Level) Logger {
	stdMu.Lock()
	defer stdMu.Unlock()
	return std.V(level)
}
