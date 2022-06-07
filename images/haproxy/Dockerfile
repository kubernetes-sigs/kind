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

# This image is a haproxy image + minimal config so the container will not exit
# while we rewrite the config at runtime and signal haproxy to reload.

ARG BASE="k8s.gcr.io/build-image/debian-base:bullseye-v1.3.0"
FROM ${BASE} as build

# NOTE: copyrights.tar.gz is a quirk of Kubernetes's debian-base image
# We extract these here so we can grab the relevant files are easily
# staged for copying into our final image.
RUN [ ! -f /usr/share/copyrights.tar.gz ] || tar -C / -xzvf /usr/share/copyrights.tar.gz

# install:
# - haproxy (see: https://haproxy.debian.net/)
# - bash (ldd is a bash script and debian-base removes bash)
# - procps (for `kill` which kind needs)
RUN apt update && \
    apt install -y --no-install-recommends haproxy=2.2.\* \
      procps bash

# copy in script for staging distro provided binary to distroless
COPY --chmod=0755 stage-binary-and-deps.sh /usr/local/bin/

# stage everything for copying into the final image
# NOTE: kind currently also uses "mkdir" and "cp" to write files within the container
# TODO: mkdir especially should be unnecessary, with a little refactoring
# NOTE: kill is used to signal haproxy to reload
ARG STAGE_DIR="/opt/stage"
RUN mkdir -p "${STAGE_DIR}" && \
    stage-binary-and-deps.sh haproxy "${STAGE_DIR}" && \
    stage-binary-and-deps.sh cp "${STAGE_DIR}" && \
    stage-binary-and-deps.sh mkdir "${STAGE_DIR}" && \
    stage-binary-and-deps.sh kill "${STAGE_DIR}" && \
    find "${STAGE_DIR}"

################################################################################

# See: https://github.com/GoogleContainerTools/distroless/tree/main/base
# This has /etc/passwd, tzdata, cacerts
FROM "gcr.io/distroless/static-debian11"

ARG STAGE_DIR="/opt/stage"

# copy staged binary + deps + copyright
COPY --from=build "${STAGE_DIR}/" /

# add our minimal config
COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg

# below roughly matches the standard haproxy image
STOPSIGNAL SIGUSR1
ENTRYPOINT ["haproxy", "-sf", "7", "-W", "-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg"]
