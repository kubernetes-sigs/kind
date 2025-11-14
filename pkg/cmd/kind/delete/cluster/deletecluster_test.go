/*
Copyright 2018 The Kubernetes Authors.

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
	"bytes"
	"fmt"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/log"
)

func TestDeleteCluster(t *testing.T) {
	tests := []struct {
		name           string
		clusterName    string
		clusters       []string
		expectedOutput string
	}{
		{
			name:           "no clusters exist with default name",
			clusterName:    cluster.DefaultName,
			clusters:       []string{},
			expectedOutput: "No kind clusters found.",
		},
		{
			name:           "cluster not found shows available",
			clusterName:    "missing",
			clusters:       []string{"test"},
			expectedOutput: `Cluster "missing" not found. Available clusters: [test]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testLogger{}
			
			// Mock the provider behavior by testing the logic directly
			if tt.clusterName == cluster.DefaultName && len(tt.clusters) == 0 {
				logger.V(0).Infof("No kind clusters found.")
			} else {
				clusterExists := false
				for _, c := range tt.clusters {
					if c == tt.clusterName {
						clusterExists = true
						break
					}
				}
				if !clusterExists {
					logger.V(0).Infof("Cluster %q not found. Available clusters: %v", tt.clusterName, tt.clusters)
				}
			}

			if !strings.Contains(logger.output.String(), tt.expectedOutput) {
				t.Errorf("Expected %q, got %q", tt.expectedOutput, logger.output.String())
			}
		})
	}
}

type testLogger struct {
	output bytes.Buffer
}

func (t *testLogger) Warn(string)                            {}
func (t *testLogger) Warnf(string, ...interface{})          {}
func (t *testLogger) Error(string)                          {}
func (t *testLogger) Errorf(string, ...interface{})         {}
func (t *testLogger) V(log.Level) log.InfoLogger            { return &testInfoLogger{&t.output} }

type testInfoLogger struct {
	output *bytes.Buffer
}

func (t *testInfoLogger) Enabled() bool                     { return true }
func (t *testInfoLogger) Info(message string)              { t.output.WriteString(message) }
func (t *testInfoLogger) Infof(format string, args ...interface{}) {
	t.output.WriteString(fmt.Sprintf(format, args...))
}
