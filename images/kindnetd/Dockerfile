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
# NOTE: tentatively follow upstream kubernetes go version based on k8s in go.mod
FROM golang:1.20
WORKDIR /go/src
# make deps fetching cacheable
COPY go.mod go.sum ./
RUN go mod download
# build
COPY . .
RUN CGO_ENABLED=0 go build -o ./kindnetd ./cmd/kindnetd

# build real kindnetd image
FROM registry.k8s.io/build-image/distroless-iptables:v0.2.2
COPY --from=0 --chown=root:root ./go/src/kindnetd /bin/kindnetd
CMD ["/bin/kindnetd"]
