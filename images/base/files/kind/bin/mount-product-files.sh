#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

# This script is a createContainer hook [1] that replicates the functionality from entrypoint script to mount product_name and product_uuid but from a product_name and product_uuid copied into the contianer rootfs to prevent all the containers from bind mounting the same file. Sharing the same bind mount between all the containers increases the latency accessing the container, preventing it from accessing in some cases.
#
# [1] https://github.com/opencontainers/runtime-spec/blob/master/config.md#createcontainer-hooks

set -o errexit
set -o nounset
set -o pipefail

# Explicitly set PATH so as not to be inherited from the container
# We have no reason to be using any binary from the container, and the
# container PATH may lack normal "host" (kind node container in this case)
# system paths.
#
# See: https://github.com/kubernetes-sigs/kind/issues/2551
#
# All of the binaries this script needs are in /usr/bin currently, but this is
# the full normal PATH for this image with pretty standard linux paths.
export PATH='/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'

# The bundle represents the dir path to container filesystem, container runtime state [1] is
# passed to the hook's stdin
#
# [1] https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#state
#
bundle=$(jq -r .bundle)

cp /kind/product_* "${bundle:?}/rootfs/"
if [[ -f /sys/class/dmi/id/product_name ]]; then
  mount -o ro,bind "${bundle:?}"/rootfs/product_name "${bundle:?}"/rootfs/sys/class/dmi/id/product_name
fi

if [[ -f /sys/class/dmi/id/product_uuid ]]; then
  mount -o ro,bind "${bundle:?}"/rootfs/product_uuid "${bundle:?}"/rootfs/sys/class/dmi/id/product_uuid
fi

if [[ -f /sys/devices/virtual/dmi/id/product_uuid ]]; then
  mount -o ro,bind "${bundle:?}"/rootfs/product_uuid "${bundle:?}"/rootfs/sys/devices/virtual/dmi/id/product_uuid
fi
