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

# Old-skool build tools.
# Simple makefile to build kind quickly and reproducibly in a container..
# Only requires docker on the host
#
# Common uses:
# - installing kind: `make install INSTALL_DIR=$HOME/go/bin`
# - building: `make build`
# - cleaning up and starting over: `make clean`

# get the repo root and output path, go_container.sh respects these
REPO_ROOT:=${CURDIR}
OUT_DIR=$(REPO_ROOT)/bin
INSTALL?=install
# make install will place binaries here
# the default path attempts to mimic go install
INSTALL_DIR?=$(shell hack/build/goinstalldir.sh)
# the output binary name, overridden when cross compiling
KIND_BINARY_NAME?=kind

# standard "make" target -> builds
all: build

# builds kind in a container, outputs to $(OUT_DIR)
kind:
	hack/build/go_container.sh go build -v -o /out/$(KIND_BINARY_NAME)

# alias for building kind
build: kind

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(KIND_BINARY_NAME) $(INSTALL_DIR)/$(KIND_BINARY_NAME)

# cleans the cache volume
clean-cache:
	docker volume rm -f kind-build-cache >/dev/null

# cleans the output directory
clean-output:
	rm -rf $(OUT_DIR)/

# standard cleanup target
clean: clean-output clean-cache

.PHONY: all kind build install clean-cache clean-output clean
