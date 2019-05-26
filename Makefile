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
GOOS?=$(shell hack/util/goos.sh)
GOARCH?=$(shell hack/util/goarch.sh)
# make install will place binaries here
# the default path attempst to mimic go install
INSTALL_DIR?=$(shell hack/util/goinstalldir.sh)
# the output binary name, overridden when cross compiling
KIND_BINARY_NAME?=kind
# use the official module proxy by default
GOPROXY?=https://proxy.golang.org
# default build image
GO_VERSION?=1.12.5
GO_IMAGE?=golang:$(GO_VERSION)
# docker volume name, used as a go module / build cache
CACHE_VOLUME?=kind-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/kind
CONTAINER_OUT_DIR=$(CONTAINER_REPO_DIR)/bin
OUT_DIR=$(REPO_ROOT)/bin
UID:=$(shell id -u)
GID:=$(shell id -g)

# standard "make" target -> builds
all: build

# creates the cache volume
make-cache:
	@echo + Ensuring build cache volume exists
	docker volume create $(CACHE_VOLUME)

# cleans the cache volume
clean-cache:
	@echo + Removing build cache volume
	docker volume rm $(CACHE_VOLUME)

# creates the output directory
out-dir:
	@echo + Ensuring build output directory exists
	mkdir -p $(OUT_DIR)

# cleans the output directory
clean-output:
	@echo + Removing build output directory
	rm -rf $(OUT_DIR)/

# builds kind in a container, outputs to $(OUT_DIR)
kind: make-cache out-dir
	@echo + Building kind binary
	docker run \
		--rm \
		-v $(CACHE_VOLUME):/go \
		-e GOCACHE=/go/cache \
		-v $(OUT_DIR):/out \
		-v $(REPO_ROOT):$(CONTAINER_REPO_DIR) \
		-w $(CONTAINER_REPO_DIR) \
		-e GO111MODULE=on \
		-e GOPROXY=$(GOPROXY) \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e HTTP_PROXY=$(HTTP_PROXY) \
		-e HTTPS_PROXY=$(HTTPS_PROXY) \
		-e NO_PROXY=$(NO_PROXY) \
		--user $(UID):$(GID) \
		$(GO_IMAGE) \
		go build -v -o /out/$(KIND_BINARY_NAME) .
	@echo + Built kind binary to $(OUT_DIR)/$(KIND_BINARY_NAME)

# alias for building kind
build: kind

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	@echo + Copying kind binary to INSTALL_DIR
	install $(OUT_DIR)/$(KIND_BINARY_NAME) $(INSTALL_DIR)/$(KIND_BINARY_NAME)

# standard cleanup target
clean: clean-cache clean-output

.PHONY: all make-cache clean-cache out-dir clean-output kind build install clean
