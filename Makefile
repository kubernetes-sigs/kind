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
GO_VERSION=1.12.5
GO_IMAGE=golang:$(GO_VERSION)
REPO_ROOT=$(PWD)
CACHE_VOLUME=kind-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/kind
CONTAINER_OUT_DIR=$(CONTAINER_REPO_DIR)/_output/bin

# standard "make" target -> builds
all: build

# creates the cache volume
make-cache:
	docker volume create $(CACHE_VOLUME)

# cleans the cache volume
clean-cache:
	docker volume rm $(CACHE_VOLUME)

# builds kind in a container, outputs to $(REPO_ROOT)/_output/bin
kind: make-cache
	docker run \
		--rm \
		-v $(CACHE_VOLUME):/go \
		-e GOCACHE=/go/cache \
		-v $(REPO_ROOT):$(CONTAINER_REPO_DIR) \
		-w $(CONTAINER_REPO_DIR) \
		$(GO_IMAGE) \
		go build -v -o $(CONTAINER_OUT_DIR)/kind .

# alias for building kind
build: kind

# standard cleanup target
clean: clean-cache

.PHONY: make-cache clean-cache kind build all clean
