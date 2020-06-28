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

import (
	"fmt"
	"testing"
)

// fakeT is a fake testing.T that tracks calls to Errorf
type fakeT int

func (t *fakeT) Errorf(format string, args ...interface{}) {
	*t++
}

func TestExpectError(t *testing.T) {
	t.Parallel()
	t.Run("expect error, nil error", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		ExpectError(&f, true, nil)
		if int(f) == 0 {
			t.Fatalf("Expected t.Errorf to be called but it was not")
		}
	})
	t.Run("expect error, have error", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		ExpectError(&f, true, fmt.Errorf("heh"))
		if int(f) != 0 {
			t.Fatalf("Expected t.Errorf not to be called but it was")
		}
	})
	t.Run("do not expect error, nil error", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		ExpectError(&f, false, nil)
		if int(f) != 0 {
			t.Fatalf("Expected t.Errorf not to be called but it was")
		}
	})
	t.Run("do not expect error, have error", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		ExpectError(&f, false, fmt.Errorf("heh"))
		if int(f) == 0 {
			t.Fatalf("Expected t.Errorf to be called but it was not")
		}
	})
}

func TestBoolEqual(t *testing.T) {
	t.Parallel()
	t.Run("not equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		BoolEqual(&f, true, false)
		if int(f) == 0 {
			t.Fatalf("Expected t.Errorf to be called but it was not")
		}
	})
	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		BoolEqual(&f, true, true)
		if int(f) != 0 {
			t.Fatalf("Expected t.Errorf not to be called but it was")
		}
	})
}

func TestStringEqual(t *testing.T) {
	t.Parallel()
	t.Run("not equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		StringEqual(&f, "a", "b")
		if int(f) == 0 {
			t.Fatalf("Expected t.Errorf to be called but it was not")
		}
	})
	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		StringEqual(&f, "a", "a")
		if int(f) != 0 {
			t.Fatalf("Expected t.Errorf not to be called but it was")
		}
	})
}

func TestDeepEqual(t *testing.T) {
	t.Parallel()
	t.Run("not equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		DeepEqual(&f, "a", "b")
		if int(f) == 0 {
			t.Fatalf("Expected t.Errorf to be called but it was not")
		}
	})
	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		var f fakeT
		DeepEqual(&f, f, f)
		if int(f) != 0 {
			t.Fatalf("Expected t.Errorf not to be called but it was")
		}
	})
}
