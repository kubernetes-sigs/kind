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

package app

import (
	"os"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind"
	"sigs.k8s.io/kind/pkg/log"
)

// Main is the kind main(), it will invoke Run(), if an error is returned
// it will then call os.Exit
func Main() {
	if err := Run(cmd.NewLogger(), cmd.StandardIOStreams()); err != nil {
		os.Exit(1)
	}
}

// Run invokes the kind root command, returning the error.
// See: sigs.k8s.io/kind/pkg/cmd/kind
func Run(logger log.Logger, streams cmd.IOStreams) error {
	return kind.NewCommand(logger, streams).Execute()
}
