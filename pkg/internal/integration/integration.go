/*
Copyright 2020 The Kubernetes Authors.

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

package integration

import "testing"

// *testing.T methods used by assert
type testingDotT interface {
	Skip(args ...interface{})
}

// MaybeSkip skips if integration tests should be skipped
// currently this is when testing.Short() is true
// This should be called at the beginning of an integration test
func MaybeSkip(t testingDotT) {
	if testing.Short() {
		t.Skip("Skipping integration test due to -short")
	}
}
