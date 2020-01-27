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

// Package load implements the `images` command
package images

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

const TagUrl = "https://registry.hub.docker.com/v1/repositories/kindest/node/tags"
const ImageName = "kindest/node"

type Message struct {
	Layer string
	Name  string
}

// NewCommand returns a new cobra.Command for images
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "images",
		Short: "show publicly available kubernetes images",
		Long:  "Displays a list of all images which can be used with cluster creation, suitable for using kind create cluster --image",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(TagUrl)

			if err != nil {
				return fmt.Errorf("Cannot get image list: %s",err)
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)

			var m []Message
			err = json.Unmarshal(body, &m)
			if err != nil {
				return fmt.Errorf("Error parsing json list of images: %s %s",TagUrl,err)
			}

			for _, t := range m {
				fmt.Printf("%s:%s\n", ImageName, t.Name)
			}

			return nil
		},
	}
	return cmd
}
