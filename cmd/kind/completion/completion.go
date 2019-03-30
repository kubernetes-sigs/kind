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

// Package completion implements the `completion` command
package completion

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kind/cmd/kind/completion/bash"
	"sigs.k8s.io/kind/cmd/kind/completion/zsh"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "completion",
		Short: "Output shell completion code for the specified shell (bash or zsh)",
		Long:  longDescription,
	}
	cmd.AddCommand(zsh.NewCommand())
	cmd.AddCommand(bash.NewCommand())
	return cmd
}

const longDescription = `
Outputs kind shell completion for the given shell (bash or zsh)
This depends on the bash-completion binary.  Example installation instructions:
# for bash users
	$ kind completion bash > ~/.kind-completion
	$ source ~/.kind-completion

# for zsh users
	% kind completion zsh > /usr/local/share/zsh/site-functions/_kind
	% autoload -U compinit && compinit

Additionally, you may want to output the completion to a file and source in your .bashrc
Note for zsh users: [1] zsh completions are only supported in versions of zsh >= 5.2
`
