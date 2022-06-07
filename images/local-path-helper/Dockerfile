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

# This image is contains the binaries needed for the local-path-provisioner
# helper pod. Currently that means: sh, rm, mkdir

ARG BASE="k8s.gcr.io/build-image/debian-base:bullseye-v1.3.0"
FROM ${BASE} as build

# NOTE: copyrights.tar.gz is a quirk of Kubernetes's debian-base image
# We extract these here so we can grab the relevant files are easily
# staged for copying into our final image.
RUN [ ! -f /usr/share/copyrights.tar.gz ] || tar -C / -xzvf /usr/share/copyrights.tar.gz

# we need bash for stage-binary-and-deps.sh
RUN apt update && apt install -y --no-install-recommends bash
# replace sh with bash
RUN ln -sf /bin/bash /bin/sh

# copy in script for staging distro provided binary to distroless
COPY --chmod=0755 stage-binary-and-deps.sh /usr/local/bin/

# local-path-provisioner needs these things for the helper pod
# TODO: we could probably coerce local-path-provisioner to use a small binary
# for these instead
ARG STAGE_DIR="/opt/stage"
RUN mkdir -p "${STAGE_DIR}" && \
    stage-binary-and-deps.sh sh "${STAGE_DIR}" && \
    stage-binary-and-deps.sh rm "${STAGE_DIR}" && \
    stage-binary-and-deps.sh mkdir "${STAGE_DIR}" && \
    find "${STAGE_DIR}"

# copy staged binary + deps + copyright into distroless
FROM "gcr.io/distroless/static-debian11"
ARG STAGE_DIR="/opt/stage"
COPY --from=build "${STAGE_DIR}/" /
