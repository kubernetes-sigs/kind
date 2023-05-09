# Copyright 2022 The Kubernetes Authors.
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

# NOTE the actual go version will be overridden
FROM golang:latest
COPY --chmod=0755 scripts/third_party/gimme/gimme /usr/local/bin/
RUN git clone --filter=tree:0 https://github.com/rancher/local-path-provisioner
ARG VERSION
# set by makefile to .go-version
ARG GO_VERSION
RUN eval "$(gimme "${GO_VERSION}")" \
    && cd local-path-provisioner \
    && git fetch && git checkout "${VERSION}" \
    && scripts/build \
    && mv bin/local-path-provisioner /usr/local/bin/local-path-provisioner \
    && GOBIN=/usr/local/bin go install github.com/google/go-licenses@latest \
    && go-licenses save --save_path=/_LICENSES .

FROM gcr.io/distroless/base-debian11
COPY --from=0 /usr/local/bin/local-path-provisioner /usr/local/bin/local-path-provisioner
COPY --from=0 /_LICENSES/* /LICENSES/
COPY --chmod=0644 files/LICENSES/* /LICENSES/*
ENTRYPOINT /usr/local/bin/local-path-provisioner
