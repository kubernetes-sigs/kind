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

package env

import (
	"io"
	"os"
	"runtime"

	isatty "github.com/mattn/go-isatty"
)

// IsTerminal returns true if the writer w is a terminal
func IsTerminal(w io.Writer) bool {
	if v, ok := (w).(*os.File); ok {
		return isatty.IsTerminal(v.Fd())
	}
	return false
}

// IsSmartTerminal returns true if the writer w is a terminal AND
// we think that the terminal is smart enough to use VT escape codes etc.
func IsSmartTerminal(w io.Writer) bool {
	if !IsTerminal(w) {
		return false
	}
	// explicitly dumb terminals are not smart
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	// On Windows WT_SESSION is set by the modern terminal component.
	// Older terminals have poor support for UTF-8, VT escape codes, etc.
	if runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "" {
		return false
	}
	return true
}
