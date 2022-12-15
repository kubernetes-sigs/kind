#!/usr/bin/env bash
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

set -o errexit -o nounset -o pipefail


: "${KIND_EXPERIMENTAL_PROVIDER:=docker}"
SSH_CONFIG=".vagrant/ssh-config"
if [ ! -f "$SSH_CONFIG" ]; then
  vagrant ssh-config > "$SSH_CONFIG"
fi

if [ "$ROOTLESS" = "rootless" ]; then
  exec ssh -F "$SSH_CONFIG" default KIND_EXPERIMENTAL_PROVIDER="$KIND_EXPERIMENTAL_PROVIDER" "${@}"
fi
exec ssh -F "$SSH_CONFIG" default sudo KIND_EXPERIMENTAL_PROVIDER="$KIND_EXPERIMENTAL_PROVIDER" "${@}"
