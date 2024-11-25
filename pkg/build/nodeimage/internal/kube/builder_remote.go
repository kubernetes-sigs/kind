/*
Copyright 2024 The Kubernetes Authors.

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
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/log"
)

var versionRE = regexp.MustCompile(`^v\d+\.\d+\.\d+`)

type remoteBuilder struct {
	version string
	logger  log.Logger
	url     string
}

var _ Builder = &remoteBuilder{}

// NewURLBuilder used to specify a complete url to a gzipped tarball
func NewURLBuilder(logger log.Logger, url string) (Builder, error) {
	// Try to parse the version from the URL as one possible way to capture what
	// we are building.
	// If building a typical community published tarball, the URL will be in the
	// format of https://dl.k8s.io/v1.30.0/kubernetes-server-linux-amd64.tar.gz
	version := ""

	parts := strings.Split(url, "/")
	for _, part := range parts {
		if versionRE.Match([]byte(part)) {
			version = part
			break
		}
	}

	return &remoteBuilder{
		version: version,
		logger:  logger,
		url:     url,
	}, nil
}

// NewReleaseBuilder used to specify a release semver and constructs a url to release artifacts
func NewReleaseBuilder(logger log.Logger, version, arch string) (Builder, error) {
	url := "https://dl.k8s.io/" + version + "/kubernetes-server-linux-" + arch + ".tar.gz"
	return &remoteBuilder{
		version: version,
		logger:  logger,
		url:     url,
	}, nil
}

// Build implements Bits.Build
func (b *remoteBuilder) Build() (Bits, error) {

	tmpDir, err := os.MkdirTemp(os.TempDir(), "k8s-tar-extract-")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory for tar extraction: %w", err)
	}

	tgzFile := filepath.Join(tmpDir, "kubernetes-"+b.version+"-server-linux-amd64.tar.gz")
	err = b.downloadURL(b.url, tgzFile)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %w", err)
	}

	err = extractTarball(tgzFile, tmpDir, b.logger)
	if err != nil {
		return nil, fmt.Errorf("error extracting tgz file: %w", err)
	}

	binDir := filepath.Join(tmpDir, "kubernetes/server/bin")

	// See if we can get an exact version, but default to the version specified
	sourceVersionRaw := b.version
	contents, err := os.ReadFile(filepath.Join(tmpDir, "kubernetes/version"))
	if err == nil {
		sourceVersionRaw = strings.TrimSpace(string(contents))
	}

	if err != nil && sourceVersionRaw == "" {
		return nil, err
	}

	return &bits{
		binaryPaths: []string{
			filepath.Join(binDir, "kubeadm"),
			filepath.Join(binDir, "kubelet"),
			filepath.Join(binDir, "kubectl"),
		},
		imagePaths: []string{
			filepath.Join(binDir, "kube-apiserver.tar"),
			filepath.Join(binDir, "kube-controller-manager.tar"),
			filepath.Join(binDir, "kube-scheduler.tar"),
			filepath.Join(binDir, "kube-proxy.tar"),
		},
		version: sourceVersionRaw,
	}, nil
}

func (b *remoteBuilder) downloadURL(url string, destPath string) error {
	output, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating file for download %q: %v", destPath, err)
	}
	defer output.Close()

	b.logger.V(0).Infof("Downloading %q", url)

	// Create a client with custom timeouts
	// to avoid idle downloads to hang the program
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
	}

	// this will stop slow downloads after 10 minutes
	// and interrupt reading of the Response.Body
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %v", err)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing HTTP fetch of %q: %v", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return fmt.Errorf("error response from %q: HTTP %v", url, response.StatusCode)
	}

	start := time.Now()
	defer func() {
		b.logger.V(2).Infof("Copying %q to %q took %q", url, destPath, time.Since(start))
	}()

	// TODO: we should add some sort of progress indicator
	_, err = io.Copy(output, response.Body)
	if err != nil {
		return fmt.Errorf("error downloading HTTP content from %q: %v", url, err)
	}
	return nil
}
