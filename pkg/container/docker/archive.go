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
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
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
		if hdr.Name == "repositories" {
			break
		}
	}
	// read and parse the tags
	b, err := ioutil.ReadAll(tr)
	if err != nil {
		return nil, err
	}
	// parse
	repoTags, err := parseRepositories(b)
	if err != nil {
		return nil, err
	}
	// convert to tags in the docker CLI sense
	res := []string{}
	for repo, tags := range repoTags {
		for tag := range tags {
			res = append(res, fmt.Sprintf("%s:%s", repo, tag))
		}
	}
	return res, nil
}

// ArchiveRepositories represents repository:tag:ref
//
// https://github.com/moby/moby/blob/master/image/spec/v1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.1.md
// https://github.com/moby/moby/blob/master/image/spec/v1.2.md
type ArchiveRepositories map[string]map[string]string

// EditArchiveRepositories applies edit to reader's repositories file
func EditArchiveRepositories(reader io.Reader, writer io.Writer, edit func(ArchiveRepositories) ArchiveRepositories) error {
	tarReader := tar.NewReader(reader)
	tarWriter := tar.NewWriter(writer)
	// iterate all entries in the tarball
	for {
		// read an entry
		hdr, err := tarReader.Next()
		if err == io.EOF {
			return tarWriter.Close()
		} else if err != nil {
			return err
		}
		b, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return err
		}

		// parse the tags if we found the repositories file
		if hdr.Name == "repositories" {
			tags, err := parseRepositories(b)
			if err != nil {
				return err
			}
			// run the user's edit function
			edit(tags)
			// marshal back the tags for writing
			b, err = json.Marshal(tags)
			if err != nil {
				return err
			}
			hdr.Size = int64(len(b))
		}

		// write to the output tarball
		if err := tarWriter.WriteHeader(hdr); err != nil {
			return err
		}
		if len(b) > 0 {
			if _, err := tarWriter.Write(b); err != nil {
				return err
			}
		}
	}
}

// returns repository:tag:ref
func parseRepositories(data []byte) (ArchiveRepositories, error) {
	var repoTags ArchiveRepositories
	if err := json.Unmarshal(data, &repoTags); err != nil {
		return nil, err
	}
	return repoTags, nil
}
