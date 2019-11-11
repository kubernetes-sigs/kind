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

package cmd

import (
	"io"
	"os"

	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/util/cli"
	"sigs.k8s.io/kind/pkg/internal/util/env"
)

// NewLogger returns the standard logger used by the kind CLI
// This logger writes to os.Stderr
func NewLogger() log.Logger {
	var writer io.Writer = os.Stderr
	if env.IsSmartTerminal(writer) {
		writer = cli.NewSpinner(writer)
	}
	return cli.NewLogger(writer, 0)
}
