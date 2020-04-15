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

package assert

import "reflect"

// *testing.T methods used by assert
type testingDotT interface {
	Errorf(format string, args ...interface{})
}

// ExpectError will call t.Errorf if expectError != (err == nil)
// t should be a *testing.T normally
func ExpectError(t testingDotT, expectError bool, err error) {
	if err != nil && !expectError {
		t.Errorf("Did not expect error: %v", err)
	}
	if err == nil && expectError {
		t.Errorf("Expected error but got none")
	}
}

// BoolEqual will call t.Errorf if expected != result
// t should be a *testing.T normally
func BoolEqual(t testingDotT, expected, result bool) {
	if expected != result {
		t.Errorf("Result did not match!")
		t.Errorf("Expected: %v", expected)
		t.Errorf("But got: %v", result)
	}
}

// StringEqual will call t.Errorf if expected != result
// t should be a *testing.T normally
func StringEqual(t testingDotT, expected, result string) {
	if expected != result {
		t.Errorf("Strings did not match!")
		t.Errorf("Expected: %q", expected)
		t.Errorf("But got: %q", result)
	}
}

// DeepEqual will call t.Errorf if !reflect.DeepEqual(expected, result)
// t should be a *testing.T normally
func DeepEqual(t testingDotT, expected, result interface{}) {
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Result did not DeepEqual Expected!")
		t.Errorf("Expected: %+v", expected)
		t.Errorf("But got: %+v", result)
	}
}
