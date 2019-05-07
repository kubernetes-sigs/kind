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

# STEP 1: Build kindnetd binary
FROM golang:1.12 AS builder
# golang envs
ARG GOARCH="amd64"
ARG GOOS=linux
ENV GO111MODULE=on
ENV CGO_ENABLED=0
# copy in sources
WORKDIR /go/src/kindnet
COPY . .
# build
RUN go get -d -v ./...
RUN go build -o /go/bin/kindnet ./cmd/kindnetd

# STEP 2: Build small image
FROM scratch
COPY --from=builder /go/bin/kindnet /bin/kindnet
CMD ["/bin/kindnet"]
