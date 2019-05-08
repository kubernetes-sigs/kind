# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Simple makefile to build kind quickly and reproducibly in a container
# Only requires docker on the host

# settings
REPO_ROOT:=${CURDIR}
# autodetect host GOOS and GOARCH by default, even if go is not installed
GOOS=$(shell hack/util/goos.sh)
GOARCH=$(shell hack/util/goarch.sh)
INSTALL_DIR=$(shell hack/util/goinstalldir.sh)
# use the official module proxy by default
GOPROXY=https://proxy.golang.org
# default build image
GO_VERSION=1.12.5
GO_IMAGE=golang:$(GO_VERSION)
# docker volume name, used as a go module / build cache
CACHE_VOLUME=kind-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/kind
CONTAINER_OUT_DIR=$(CONTAINER_REPO_DIR)/_output/bin
HOST_OUT_DIR=$(REPO_ROOT)/_output/bin

# standard "make" target -> builds
all: build

# creates the cache volume
make-cache:
	docker volume create $(CACHE_VOLUME)

# cleans the cache volume
clean-cache:
	docker volume rm $(CACHE_VOLUME)

# creates the output directory
out-dir:
	mkdir -p $(REPO_ROOT)/_output/bin

# cleans the output directory
clean-output:
	rm -rf $(REPO_ROOT)/_output

# builds kind in a container, outputs to $(REPO_ROOT)/_output/bin
kind: make-cache out-dir
	docker run \
		--rm \
		-v $(CACHE_VOLUME):/go \
		-e GOCACHE=/go/cache \
		-v $(REPO_ROOT):$(CONTAINER_REPO_DIR) \
		-w $(CONTAINER_REPO_DIR) \
		-e GO111MODULE=on \
		-e GOPROXY=$(GOPROXY) \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		$(GO_IMAGE) \
		go build -v -o $(CONTAINER_OUT_DIR)/kind .

# alias for building kind
build: kind

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	cp $(HOST_OUT_DIR)/kind $(INSTALL_DIR)/kind

# standard cleanup target
clean: clean-cache clean-output

.PHONY: all make-cache clean-cache out-dir clean-output kind build install clean
