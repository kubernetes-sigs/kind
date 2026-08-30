/*
Copyright 2021 The Kubernetes Authors.

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

package sets

import (
	"testing"
)

func TestString(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		s := NewString("a", "b", "c")
		if s.Len() != 3 {
			t.Errorf("Expected len 3, got %d", s.Len())
		}
		if !s.Has("a") || !s.Has("b") || !s.Has("c") {
			t.Errorf("Set is missing expected elements: %v", s)
		}
		
		if s.Has("d") {
			t.Errorf("Set contains unexpected element 'd'")
		}

		s.Insert("d")
		if !s.Has("d") || s.Len() != 4 {
			t.Errorf("Insert failed, set: %v", s)
		}

		s.Delete("a", "b")
		if s.Has("a") || s.Has("b") || s.Len() != 2 {
			t.Errorf("Delete failed, set: %v", s)
		}
	})

	t.Run("Union & Intersection", func(t *testing.T) {
		s1 := NewString("a", "b", "c")
		s2 := NewString("b", "c", "e", "d")

		diff := s2.Difference(s1)
		if diff.Len() != 2 || !diff.Has("e") || !diff.Has("d") {
			t.Errorf("Difference failed, expected {\"d\", \"e\"}, got %v", diff)
		}

		union := s1.Union(s2)
		if union.Len() != 5 || !union.HasAll("a", "b", "c", "d", "e") {
			t.Errorf("Union failed, got %v, expected {\"a\", \"b\", \"c\", \"d\", \"e\"}", union)
		}

		intersect := s1.Intersection(s2)
		if intersect.Len() != 2 || !intersect.HasAll("b", "c") {
			t.Errorf("Intersection failed, got %v, expected {\"b\", \"c\"}", intersect)
		}

		if !union.IsSuperset(s1) {
			t.Errorf("IsSuperset failed, union should be superset of s1")
		}
		if s1.IsSuperset(union) {
			t.Errorf("IsSuperset failed, s1 should not be superset of union")
		}

		s3 := NewString("a", "b", "c")

		if !s1.Equal(s3) {
			t.Errorf("Equal Failed, s1 should be equal to s3")
		}
		if s2.Equal(s3) {
			t.Errorf("Equal Failed, s2 should not be equal to s3")
		}
	})

	t.Run("Queries", func(t *testing.T) {
		s := NewString("a", "b", "c")

		if !s.HasAll("a", "b", "c") {
			t.Errorf("HasAll failed, s: %v", s)
		}
		if !s.HasAll("a", "b") {
			t.Errorf("HasAll failed for existing elements")
		}
		if s.HasAll("a", "d") {
			t.Errorf("HasAll should return false if one element is missing")
		}

		if !s.HasAny("a", "d") {
			t.Errorf("HasAny failed for partially existing elements")
		}
		if s.HasAny("e", "d") {
			t.Errorf("HasAny failed as it return true even if no element exists")
		}
	})

	t.Run("Lists", func(t *testing.T) {
		s := NewString("c", "a", "b")

		list := s.List()
		if len(list) != 3 || list[0] != "a" || list[1] != "b" || list[2] != "c" {
			t.Errorf("List failed or not sorted, got: %v", list)
		}

		unsorted := s.UnsortedList()
		if len(unsorted) != 3 {
			t.Errorf("UnsortedList returned wrong length, got: %v", unsorted)
		}

		val, ok := s.PopAny()
		if !ok || val == "" || s.Len() != 2 {
			t.Errorf("PopAny failed, got val: %s, ok: %v, len: %d", val, ok, s.Len())
		}

		empty := NewString()
		_, ok = empty.PopAny()
		if ok {
			t.Errorf("PopAny on empty set should return false")
		}
	})

	t.Run("StringKeySet", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		s := StringKeySet(m)
		if s.Len() != 2 || !s.HasAll("a", "b") {
			t.Errorf("StringKeySet failed, got: %v", s)
		}

		// Test the panic path
		t.Run("panics on non-map input", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected StringKeySet to panic when given a non-map, but it did not")
				}
			}()
			StringKeySet(123)
		})
	})
}