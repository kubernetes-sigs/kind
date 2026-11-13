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

package cluster

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

const testConfig = `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
`

func TestConfigOptionURL(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		_, _ = w.Write([]byte(testConfig))
	}))
	defer server.Close()

	_, err := configOption(server.URL+"/kind-config.yaml", nil)
	assert.ExpectError(t, false, err)
	mu.Lock()
	defer mu.Unlock()
	assert.DeepEqual(t, []string{"GET /kind-config.yaml"}, requests)
}

func TestConfigOptionURLError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, err := configOption(server.URL+"/kind-config.yaml", nil)
	assert.ExpectError(t, true, err)
}

func TestConfigOptionInvalidURL(t *testing.T) {
	t.Parallel()
	_, err := configOption("https://", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid config URL") {
		t.Errorf("expected an invalid config URL error, got: %v", err)
	}
}

func TestConfigOptionFilePath(t *testing.T) {
	t.Parallel()
	// config files are not read until the cluster is created, so these
	// should not error even though they do not exist, unless fetched as URLs
	for _, path := range []string{"kind-config.yaml", "http-kind-config.yaml"} {
		_, err := configOption(path, nil)
		assert.ExpectError(t, false, err)
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name        string
		URL         string
		ExpectError bool
	}{
		{
			Name: "https",
			URL:  "https://example.com/kind-config.yaml",
		},
		{
			Name: "http with port",
			URL:  "http://127.0.0.1:8080/kind-config.yaml",
		},
		{
			Name:        "missing host",
			URL:         "https://",
			ExpectError: true,
		},
		{
			Name:        "missing host with path",
			URL:         "https:///kind-config.yaml",
			ExpectError: true,
		},
		{
			Name:        "port without host",
			URL:         "https://:8080/kind-config.yaml",
			ExpectError: true,
		},
		{
			Name:        "invalid host",
			URL:         "https://exa mple.com/kind-config.yaml",
			ExpectError: true,
		},
		{
			Name:        "invalid port",
			URL:         "https://example.com:port/kind-config.yaml",
			ExpectError: true,
		},
		{
			Name:        "invalid escape",
			URL:         "https://example.com/%zz",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		c := c // capture loop variable
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			assert.ExpectError(t, c.ExpectError, validateURL(c.URL))
		})
	}
}

func TestFetchURL(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kind-config.yaml" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(testConfig))
	}))
	defer server.Close()
	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServer.Close()

	cases := []struct {
		Name        string
		URL         string
		Expected    string
		ExpectError bool
	}{
		{
			Name:     "config",
			URL:      server.URL + "/kind-config.yaml",
			Expected: testConfig,
		},
		{
			Name:        "not found",
			URL:         server.URL + "/missing.yaml",
			ExpectError: true,
		},
		{
			Name:        "unreachable",
			URL:         closedServer.URL + "/kind-config.yaml",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		c := c // capture loop variable
		t.Run(c.Name, func(t *testing.T) {
			raw, err := fetchURL(c.URL)
			assert.ExpectError(t, c.ExpectError, err)
			assert.StringEqual(t, c.Expected, string(raw))
		})
	}
}
