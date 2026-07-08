/*
Copyright 2020 The Kubernetes Authors.

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

package nodeimage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/version"
	"sigs.k8s.io/kind/pkg/log"
)

// Build builds a node image using the supplied options
func Build(options ...Option) error {
	// default options
	ctx := &buildContext{
		image:     DefaultImage,
		baseImage: DefaultBaseImage,
		logger:    log.NoopLogger{},
		arch:      runtime.GOARCH,
	}

	// apply user options
	for _, option := range options {
		if err := option.apply(ctx); err != nil {
			return err
		}
	}

	// verify that we're using a supported arch
	if !supportedArch(ctx.arch) {
		ctx.logger.Warnf("unsupported architecture %q", ctx.arch)
	}

	if ctx.buildType == "" {
		ctx.buildType = detectBuildType(ctx.kubeParam)
		if ctx.buildType != "" {
			ctx.logger.V(0).Infof("Detected build type: %q", ctx.buildType)
		}
	}

	if ctx.buildType == "url" {
		ctx.logger.V(0).Infof("Building using URL: %q", ctx.kubeParam)
		builder, err := kube.NewURLBuilder(ctx.logger, ctx.kubeParam)
		if err != nil {
			return err
		}
		ctx.builder = builder
	}

	if ctx.buildType == "file" {
		ctx.logger.V(0).Infof("Building using local file: %q", ctx.kubeParam)
		if info, err := os.Stat(ctx.kubeParam); err == nil && info.Mode().IsRegular() {
			builder, err := kube.NewTarballBuilder(ctx.logger, ctx.kubeParam)
			if err != nil {
				return err
			}
			ctx.builder = builder
		}
	}

	if ctx.buildType == "release" {
		ctx.logger.V(0).Infof("Building using release %q artifacts", ctx.kubeParam)
		kubever, err := version.ParseSemantic(ctx.kubeParam)
		if err == nil {
			builder, err := kube.NewReleaseBuilder(ctx.logger, "v"+kubever.String(), ctx.arch)
			if err != nil {
				return err
			}
			ctx.builder = builder
		} else {
			if _, err := os.Stat(ctx.kubeParam); err != nil {
				ctx.logger.V(0).Infof("%s is not a valid kubernetes version", ctx.kubeParam)
				return fmt.Errorf("%s is not a valid kubernetes version", ctx.kubeParam)
			}
		}
	}

	if ctx.buildType == "ci" {
		ctx.logger.V(0).Infof("Building using CI %q artifacts", ctx.kubeParam)
		marker := ctx.kubeParam
		if marker == "" {
			marker = "latest"
		}
		if !strings.HasPrefix(marker, "ci/") {
			marker = "ci/" + marker
		}
		if !strings.HasSuffix(marker, ".txt") {
			marker = marker + ".txt"
		}
		markerURL := "https://dl.k8s.io/" + marker
		resolvedVersion, err := fetchURL(ctx.logger, markerURL)
		if err != nil {
			return errors.Wrapf(err, "error resolving CI version from %s", markerURL)
		}
		ctx.logger.V(0).Infof("Resolved CI version: %s", resolvedVersion)
		builder, err := kube.NewCIBuilder(ctx.logger, resolvedVersion, ctx.arch)
		if err != nil {
			return err
		}
		ctx.builder = builder
	}

	if ctx.builder == nil {
		// locate sources if no kubernetes source was specified
		if ctx.kubeParam == "" {
			kubeRoot, err := kube.FindSource()
			if err != nil {
				return errors.Wrap(err, "error finding kuberoot")
			}
			ctx.kubeParam = kubeRoot
		}
		ctx.logger.V(0).Infof("Building using source: %q", ctx.kubeParam)

		// initialize bits
		builder, err := kube.NewDockerBuilder(ctx.logger, ctx.kubeParam, ctx.arch)
		if err != nil {
			return err
		}
		ctx.builder = builder
	}

	// do the actual build
	return ctx.Build()
}

// detectBuildType detect the type of build required based on the param passed in the following order
// url: if the param is a valid http or https url
// file: if the param refers to an existing regular file
// source: if the param refers to an existing directory
// ci: if the param starts with "ci/" (indicates a CI build version marker)
// release: if the param is a semantic version expression (does this require the v preprended?
func detectBuildType(param string) string {
	u, err := url.ParseRequestURI(param)
	if err == nil {
		if u.Scheme == "http" || u.Scheme == "https" {
			return "url"
		}
	}
	if info, err := os.Stat(param); err == nil {
		if info.Mode().IsRegular() {
			return "file"
		}
		if info.Mode().IsDir() {
			return "source"
		}
	}
	if strings.HasPrefix(param, "ci/") {
		return "ci"
	}
	_, err = version.ParseSemantic(param)
	if err == nil {
		return "release"
	}
	return ""
}

func supportedArch(arch string) bool {
	switch arch {
	default:
		return false
	// currently we nominally support building node images for these
	case "amd64":
	case "arm64":
	}
	return true
}

func fetchURL(logger log.Logger, url string) (string, error) {
	logger.V(0).Infof("Fetching %q", url)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
