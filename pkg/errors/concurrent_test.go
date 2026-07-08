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
	"sort"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestUntilErrorConcurrent(t *testing.T) {
	t.Parallel()
	t.Run("first to return error", func(t *testing.T) {
		t.Parallel()
		// test that the first function to return an error is returned
		expected := New("first")
		wait := make(chan bool)
		result := UntilErrorConcurrent([]func() error{
			func() error {
				<-wait
				return New("second")
			},
			func() error {
				return expected
			},
		})
		wait <- true
		assert.DeepEqual(t, expected, result)
	})
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		result := UntilErrorConcurrent([]func() error{
			func() error {
				return nil
			},
		})
		var expected error
		assert.DeepEqual(t, expected, result)
	})
}

func TestAggregateConcurrent(t *testing.T) {
	t.Parallel()
	t.Run("all errors returned", func(t *testing.T) {
		t.Parallel()
		// test that the first function to return an error is returned
		first := New("first")
		second := New("second")
		expected := []error{first, second}
		result := AggregateConcurrent([]func() error{
			func() error {
				return second
			},
			func() error {
				return first
			},
		})
		resultErrors := Errors(result)
		// We just want to check if we aggregate all the errors independent of the order
		sort.SliceStable(resultErrors, func(i, j int) bool {
			return resultErrors[i].Error() < resultErrors[j].Error()
		})
		assert.DeepEqual(t, expected, resultErrors)
	})
	t.Run("one error", func(t *testing.T) {
		t.Parallel()
		expected := New("foo")
		result := AggregateConcurrent([]func() error{
			func() error {
				return expected
			},
		})
		assert.DeepEqual(t, expected, result)
	})
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		result := AggregateConcurrent([]func() error{
			func() error {
				return nil
			},
		})
		var expected error
		assert.DeepEqual(t, expected, result)
	})
}
