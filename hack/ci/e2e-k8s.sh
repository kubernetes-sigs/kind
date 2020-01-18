#!/bin/sh
# Copyright 2018 The Kubernetes Authors.
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

# hack script for running a kind e2e
# must be run with a kubernetes checkout in $PWD (IE from the checkout)
# Usage: SKIP="ginkgo skip regex" FOCUS="ginkgo focus regex" kind-e2e.sh

set -o errexit -o nounset -o xtrace

# Settings:
# SKIP: ginkgo skip regex
# FOCUS: ginkgo focus regex
# BUILD_TYPE: bazel or docker
#
export KIND_BUILD_TYPE="${BUILD_TYPE:-bazel}"
# when neither of tehse are set we need the old default
# TODO: eliminate this script and upadte the prowjobs do do these explicitly
if [ -z "${FOCUS:-}${SKIP:-}" ]; then
  kind test conformance --ip-family="${IP_FAMILY:-ipv4}"
else
  export KIND_TEST_FOCUS_OVERRIDE="${FOCUS:-}"
  export KIND_TEST_SKIP_OVERRIDE="${SKIP:-}"
  kind test presubmit --ip-family="${IP_FAMILY:-ipv4}"
fi
