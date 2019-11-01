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
)

func TestErrors(t *testing.T) {
	t.Parallel()
	t.Run("wrapped aggregate", func(t *testing.T) {
		t.Parallel()
		errs := []error{New("foo"), Errorf("bar")}
		err := Wrapf(NewAggregate(errs), "baz: %s", "quux")
		result := Errors(err)
		if !reflect.DeepEqual(result, errs) {
			t.Errorf("Result did not equal Expected")
			t.Errorf("Expected: %v", errs)
			t.Errorf("Result: %v", result)
		}
	})
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		result := Errors(nil)
		var expected []error
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Result did not equal Expected")
			t.Errorf("Expected: %v", expected)
			t.Errorf("Result: %v", result)
		}
	})
}
