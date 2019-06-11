#!/usr/bin/env bash
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

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"


# place to stick temp binaries
BINDIR="${REPO_ROOT}/bin"
mkdir -p "${BINDIR}"

# TMP_REPO is used in make_temp_repo_copy
TMP_REPO="$(TMPDIR="${BINDIR}" mktemp -d "${BINDIR}/verify-deps.XXXXX")"

# exit trap cleanup for TMP_REPO
cleanup() {
  if [[ -n "${TMP_REPO}" ]]; then
    rm -rf "${TMP_REPO}"
  fi
}

# copies repo into a temp root saved to TMP_REPO
make_temp_repo_copy() {
  # we need to copy everything but bin (and the old _output) (which is .gitignore anyhow)
  find . \
    -mindepth 1 -maxdepth 1 \
    \( \
      -type d -path "./.git" -o \
      -type d -path "./bin" -o \
      -type d -path "./_output" \
    \) -prune -o \
    -exec bash -c 'cp -r "${0}" "${1}/${0}" >/dev/null 2>&1' {} "${TMP_REPO}" \;
}

main() {
  trap cleanup EXIT

  # copy repo root into tempdir under ./_output
  make_temp_repo_copy

  # run generated code update script
  cd "${TMP_REPO}"
  REPO_ROOT="${TMP_REPO}" hack/update/generated.sh

  # make sure the temp repo has no changes relative to the real repo
  diff=$(diff -Nupr \
          -x ".git" \
          -x "bin" \
          -x "_output" \
          -x "vendor" \
         "${REPO_ROOT}" "${TMP_REPO}" 2>/dev/null || true)
  if [[ -n "${diff}" ]]; then
    echo "unexpectedly dirty working directory after hack/update/generated.sh" >&2
    echo "" >&2
    echo "${diff}" >&2
    echo "" >&2
    echo "please run hack/update/generated.sh" >&2
    exit 1
  fi
}

main
