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

// Package load implements the `load` command
package load

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name  string
	Nodes []string
}

var (
	shaKey = "@sha256:"
)

// NewCommand returns a new cobra.Command for loading an image into a cluster
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("name of image is required")
			}
			return nil
		},
		Use:   "docker-image",
		Short: "loads docker image from host into nodes",
		Long:  "loads docker image from host into all or specified nodes by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.Name,
		"name",
		cluster.DefaultName,
		"the cluster context name",
	)
	cmd.Flags().StringSliceVar(
		&flags.Nodes,
		"nodes",
		nil,
		"comma separated list of nodes to load images into",
	)
	return cmd
}

func runE(logger log.Logger, flags *flagpole, args []string) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)

	// Check that the image exists locally and gets its ID, if not return error
	imageName := args[0]
	imageID, err := imageID(imageName)
	if err != nil {
		return fmt.Errorf("image: %q not present locally", imageName)
	}

	// Check if the cluster nodes exist
	nodeList, err := provider.ListInternalNodes(flags.Name)
	if err != nil {
		return err
	}
	if len(nodeList) == 0 {
		return fmt.Errorf("no nodes found for cluster %q", flags.Name)
	}

	// map cluster nodes by their name
	nodesByName := map[string]nodes.Node{}
	for _, node := range nodeList {
		// TODO(bentheelder): this depends on the fact that ListByCluster()
		// will have name for nameOrId.
		nodesByName[node.String()] = node
	}

	// pick only the user selected nodes and ensure they exist
	// the default is all nodes unless flags.Nodes is set
	candidateNodes := nodeList
	if len(flags.Nodes) > 0 {
		candidateNodes = []nodes.Node{}
		for _, name := range flags.Nodes {
			node, ok := nodesByName[name]
			if !ok {
				return fmt.Errorf("unknown node: %q", name)
			}
			candidateNodes = append(candidateNodes, node)
		}
	}

	// pick only the nodes that don't have the image
	selectedNodes := []nodes.Node{}
	tagName := imageName
	strs := strings.Split(imageName, shaKey)
	if len(strs) > 0 {
		tagName = strs[0]
	}
	for _, node := range candidateNodes {
		id, err := nodeutils.ImageID(node, tagName)
		if err != nil || id != imageID {
			selectedNodes = append(selectedNodes, node)
			logger.V(0).Infof("Image: %q with ID %q not present on node %q", imageName, imageID, node.String())
		}
	}

	if len(selectedNodes) == 0 {
		return nil
	}

	// Save the image into a tar
	dir, err := fs.TempDir("", "image-tar")
	if err != nil {
		return errors.Wrap(err, "failed to create tempdir")
	}
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = save(imageName, imageTarPath)
	if err != nil {
		return err
	}

	// Load the image on the selected nodes
	fns := []func() error{}
	for _, selectedNode := range selectedNodes {
		selectedNode := selectedNode // capture loop variable
		fns = append(fns, func() error {
			return loadImage(imageTarPath, selectedNode)
		})
	}
	return errors.UntilErrorConcurrent(fns)
}

// TODO: we should consider having a cluster method to load images

// loads an image tarball onto a node
func loadImage(imageTarName string, node nodes.Node) error {
	f, err := os.Open(imageTarName)
	if err != nil {
		return errors.Wrap(err, "failed to open image")
	}
	defer f.Close()
	return nodeutils.LoadImageArchive(node, f)
}

// save saves image to dest, as in `docker save`
func save(image, dest string) error {
	tag := image
	if !strings.Contains(tag, shaKey) {
		return exec.Command("docker", "save", "-o", dest, image).Run()
	}

	// docker save does not write tag when sha256 is specified
	strs := strings.Split(tag, shaKey)
	if len(strs) > 0 {
		tag = strs[0]
	}

	tarFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer tarFile.Close()
	tw := tar.NewWriter(tarFile)
	defer tw.Close()
	saveCmd := exec.Command("docker", "save", image)
	var b bytes.Buffer
	saveCmd.SetStdout(&b)
	err = saveCmd.Run()
	if err != nil {
		return err
	}

	tr := tar.NewReader(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		var content bytes.Buffer
		if _, err := io.Copy(&content, tr); err != nil {
			return err
		}
		if hdr.Name == "manifest.json" {
			s := content.String()
			s = strings.Replace(s, "\"RepoTags\":null", "\"RepoTags\":[\""+tag+"\"]", -1)
			hdr.Size = int64(len([]byte(s)))
			err = tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
			_, err = tw.Write([]byte(s))
			if err != nil {
				return err
			}
		} else {
			err = tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
			_, err = tw.Write(content.Bytes())
			if err != nil {
				return err
			}
		}
	}
	tw.Flush()
	return nil
}

// imageID return the Id of the container image
func imageID(containerNameOrID string) (string, error) {
	cmd := exec.Command("docker", "image", "inspect",
		"-f", "{{ .Id }}",
		containerNameOrID, // ... against the container
	)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return "", err
	}
	if len(lines) != 1 {
		return "", errors.Errorf("Docker image ID should only be one line, got %d lines", len(lines))
	}
	return lines[0], nil
}
