/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

func TestRequiredNodeImages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cluster *config.Cluster
		want    sets.String
	}{
		{
			name: "Cluster with different images",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				n, n2 := config.Node{}, config.Node{}
				n.Image = "node1"
				n2.Image = "node2"
				c.Nodes = []config.Node{n, n2}
				return &c
			}(),
			want: sets.NewString("node1", "node2"),
		},
		{
			name: "Cluster with nodes with same image",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				n, n2 := config.Node{}, config.Node{}
				n.Image = "node1"
				n2.Image = "node1"
				c.Nodes = []config.Node{n, n2}
				return &c
			}(),
			want: sets.NewString("node1"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RequiredNodeImages(tt.cluster); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequiredNodeImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
