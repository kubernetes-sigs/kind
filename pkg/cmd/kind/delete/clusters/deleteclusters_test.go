/*
Copyright The Kubernetes Authors.

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

package clusters

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/log"
)

type fakeProvider struct {
	clusters   []string
	listErr    error
	deleteErrs map[string]error
	deleted    []string
}

func (p *fakeProvider) List() ([]string, error) {
	return p.clusters, p.listErr
}

func (p *fakeProvider) Delete(name, explicitKubeconfigPath string) error {
	p.deleted = append(p.deleted, name)
	return p.deleteErrs[name]
}

func TestDeleteClustersWithProvider(t *testing.T) {
	t.Parallel()

	t.Run("returns failures after attempting every cluster", func(t *testing.T) {
		t.Parallel()
		provider := &fakeProvider{
			deleteErrs: map[string]error{
				"failed": fmt.Errorf("runtime unavailable"),
			},
		}

		err := deleteClustersWithProvider(
			log.NoopLogger{}, provider, &flagpole{}, []string{"failed", "succeeded"},
		)
		if err == nil {
			t.Fatal("expected deletion failure")
		}
		if !strings.Contains(err.Error(), `failed to delete cluster "failed": runtime unavailable`) {
			t.Fatalf("unexpected error: %v", err)
		}
		if expected := []string{"failed", "succeeded"}; !reflect.DeepEqual(provider.deleted, expected) {
			t.Fatalf("expected deletion attempts %q, got %q", expected, provider.deleted)
		}
	})

	t.Run("returns nil when every cluster is deleted", func(t *testing.T) {
		t.Parallel()
		provider := &fakeProvider{}

		if err := deleteClustersWithProvider(log.NoopLogger{}, provider, &flagpole{}, []string{"one", "two"}); err != nil {
			t.Fatalf("unexpected deletion error: %v", err)
		}
		if expected := []string{"one", "two"}; !reflect.DeepEqual(provider.deleted, expected) {
			t.Fatalf("expected deletion attempts %q, got %q", expected, provider.deleted)
		}
	})
}
