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

// Package docker contains helpers for working with docker
// This package has no stability guarantees whatsoever!
package docker

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/kind/pkg/errors"
)

// GetArchiveTags obtains a list of "repo:tag" docker image tags from a
// given docker image archive (tarball) path
// compatible with all known specs:
// https://github.com/moby/moby/blob/master/image/spec/v1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.2.md
func GetArchiveTags(path string) ([]string, error) {
	// open the archive and find the repositories entry
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	var hdr *tar.Header
	for {
		hdr, err = tr.Next()
		if err == io.EOF {
			return nil, errors.New("could not find image metadata")
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "manifest.json" || hdr.Name == "repositories" {
			break
		}
	}
	// read and parse the tags
	b, err := io.ReadAll(tr)
	if err != nil {
		return nil, err
	}
	res := []string{}
	// parse
	if hdr.Name == "repositories" {
		repoTags, err := parseRepositories(b)
		if err != nil {
			return nil, err
		}
		// convert to tags in the docker CLI sense
		for repo, tags := range repoTags {
			for tag := range tags {
				res = append(res, fmt.Sprintf("%s:%s", repo, tag))
			}
		}
	} else if hdr.Name == "manifest.json" {
		manifest, err := parseDockerV1Manifest(b)
		if err != nil {
			return nil, err
		}
		res = append(res, manifest[0].RepoTags...)
	}
	return res, nil
}

// archiveRepositories represents repository:tag:ref
//
// https://github.com/moby/moby/blob/master/image/spec/v1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.2.md
type archiveRepositories map[string]map[string]string

// https://github.com/moby/moby/blob/master/image/spec/v1.2.md#combined-image-json--filesystem-changeset-format
type metadataEntry struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// returns repository:tag:ref
func parseRepositories(data []byte) (archiveRepositories, error) {
	var repoTags archiveRepositories
	if err := json.Unmarshal(data, &repoTags); err != nil {
		return nil, err
	}
	return repoTags, nil
}

// parseDockerV1Manifest parses Docker Image Spec v1 manifest (not OCI Image Spec manifest)
// https://github.com/moby/moby/blob/v20.10.22/image/spec/v1.2.md#combined-image-json--filesystem-changeset-format
func parseDockerV1Manifest(data []byte) ([]metadataEntry, error) {
	var entries []metadataEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
