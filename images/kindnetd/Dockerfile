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

# first stage build kindnetd binary
# NOTE: the actual go version will be overridden
FROM --platform=$BUILDPLATFORM docker.io/library/golang:latest
WORKDIR /go/src
COPY --chmod=0755 scripts/third_party/gimme/gimme /usr/local/bin/
# make deps fetching cacheable
COPY go.mod go.sum ./
# set by makefile to .go-version
ARG GO_VERSION
RUN eval "$(gimme "${GO_VERSION}")" \
    && go mod download \
    && GOBIN=/usr/local/bin go install github.com/google/go-licenses@latest
# build
COPY . .
ARG TARGETARCH
RUN eval "$(gimme "${GO_VERSION}")" \
    && CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o ./kindnetd ./cmd/kindnetd \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES ./cmd/kindnetd

# build real kindnetd image
FROM registry.k8s.io/build-image/distroless-iptables:v0.2.3
COPY --from=0 --chown=root:root ./go/src/kindnetd /bin/kindnetd
COPY --from=0 /_LICENSES/* /LICENSES/
COPY --chmod=0644 files/LICENSES/* /LICENSES/*
CMD ["/bin/kindnetd"]
