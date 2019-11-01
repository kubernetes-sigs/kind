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
	"bytes"
	"io/ioutil"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// test trivial nil case
	if IsTerminal(nil) {
		t.Fatalf("IsTerminal should be false for nil Writer")
	}
	// test something that isn't even a file
	var buff bytes.Buffer
	if IsTerminal(&buff) {
		t.Fatalf("IsTerminal should be false for bytes.Buffer")
	}
	// test a file
	f, err := ioutil.TempFile("", "kind-isterminal")
	if err != nil {
		t.Fatalf("Failed to create tempfile %v", err)
	}
	if IsTerminal(f) {
		t.Fatalf("IsTerminal should be false for nil Writer")
	}
	// TODO: testing an actual PTY would be somewhat tricky to do cleanly
	// but we should maybe do this in the future.
	// At least we know this doesn't trigger on things that are obviously not
	// terminals
}
