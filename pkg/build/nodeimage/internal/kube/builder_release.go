/*
Copyright 2021 The Kubernetes Authors.

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

package kube

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

// releaseBuilder uses pre-built artifacts
type releaseBuilder struct {
	releaseUrl string
	arch       string
	logger     log.Logger
}

var _ Builder = &releaseBuilder{}

// NewReleaseBuilder returns a new Bits from a release URL.
func NewReleaseBuilder(logger log.Logger, kubeBase, releaseUrl, arch string) (Builder, error) {
	return &releaseBuilder{
		releaseUrl: releaseUrl,
		arch:       arch,
		logger:     logger,
	}, nil
}

func (b *releaseBuilder) Build() (Bits, error) {
	u, err := url.Parse(b.releaseUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing release-url")
	}
	sourceVersion := filepath.Base(u.EscapedPath())

	tmpdir, err := ioutil.TempDir("", "kind-release-fetch")
	if err != nil {
		return nil, err
	}
	b.logger.V(10).Infof("Created tmp dir %s for release downloads", tmpdir)

	binPaths, err := b.getReleaseFiles(tmpdir, u, []string{"kubeadm", "kubelet", "kubectl"})
	if err != nil {
		return nil, err
	}

	imagePaths, err := b.getReleaseFiles(tmpdir, u, []string{
		"kube-apiserver.tar",
		"kube-controller-manager.tar",
		"kube-scheduler.tar",
		"kube-proxy.tar",
	})
	if err != nil {
		return nil, err
	}

	return &bits{
		binaryPaths: binPaths,
		imagePaths:  imagePaths,
		version:     sourceVersion,
	}, nil
}

func (b *releaseBuilder) getReleaseFiles(tmpdir string, baseUrl *url.URL, files []string) (paths []string, err error) {
	for _, f := range files {
		dlUrl := *baseUrl
		dlUrl.Path = filepath.Join(dlUrl.Path, "bin/linux", b.arch, f)
		dlpath := filepath.Join(tmpdir, f)
		b.logger.V(10).Infof("Downloading %s to %s", dlUrl.String(), dlpath)
		err = getReleaseContent(dlpath, dlUrl.String())
		if err != nil {
			return nil, err
		}
		paths = append(paths, dlpath)
	}
	return paths, nil

}

func getReleaseContent(file, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "error getting file %s", file)
	}
	defer resp.Body.Close()

	dst, err := os.Create(file)
	if err != nil {
		return errors.Wrapf(err, "error creating temporary file %s", file)
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	return errors.Wrapf(err, "error writing content to %s", file)
}
