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

package errors

import (
	"reflect"
	"testing"

	pkgerrors "github.com/pkg/errors"
)

func TestStackTrace(t *testing.T) {
	t.Parallel()
	t.Run("wrapped chain", func(t *testing.T) {
		t.Parallel()
		err := New("foo")
		expected := err.(StackTracer).StackTrace()
		result := StackTrace(Wrap(Wrap(err, "bar"), "baz"))
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Result did not equal Expected")
			t.Errorf("Expected: %v", expected)
			t.Errorf("Result: %v", result)
		}
	})
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		result := StackTrace(nil)
		var expected pkgerrors.StackTrace
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Result did not equal Expected")
			t.Errorf("Expected: %v", expected)
			t.Errorf("Result: %v", result)
		}
	})
}
