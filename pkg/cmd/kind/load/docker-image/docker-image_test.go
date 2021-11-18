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

package load

import (
	"reflect"
	"sort"
	"testing"
)

func Test_removeDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		want  []string
	}{
		{
			name:  "empty",
			slice: []string{},
			want:  []string{},
		},
		{
			name:  "all different",
			slice: []string{"one", "two"},
			want:  []string{"one", "two"},
		},
		{
			name:  "one dup",
			slice: []string{"one", "two", "two"},
			want:  []string{"one", "two"},
		},
		{
			name:  "two dup",
			slice: []string{"one", "two", "two", "one"},
			want:  []string{"one", "two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeDuplicates(tt.slice)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeDuplicates() = %v, want %v", got, tt.want)
			}
		})
	}
}
