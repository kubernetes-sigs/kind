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
BINDIR="${REPO_ROOT}/_output/bin"
mkdir -p "${BINDIR}"

# TMP_GOPATH is used in make_temp_root
TMP_GOPATH="$(TMPDIR="${BINDIR}" mktemp -d "${BINDIR}/verify-deps.XXXXX")"

# exit trap cleanup for TMP_GOPATH
cleanup() {
  if [[ -n "${TMP_GOPATH}" ]]; then
    rm -rf "${TMP_GOPATH}"
  fi
}

# copies repo into a temp root relative to TMP_GOPATH
make_temp_root() {
  # make a fake gopath
  local fake_root="${TMP_GOPATH}/src/sigs.k8s.io/kind"
  mkdir -p "${fake_root}/.."
  # we need to copy everything but _output (which is .gitignore anyhow)
  find . \
    -type d -path "./_output" -prune -o \
    -maxdepth 1 -mindepth 1 -exec cp -a {} "${fake_root}/{}" \;
}

main() {
  trap cleanup EXIT

  # copy repo root into tempdir under ./_output
  echo "Copying tree into temp root ..."
  make_temp_root
  local fake_root="${TMP_GOPATH}/src/sigs.k8s.io/kind"

  # run vendor update script
  echo "Updating deps in '${fake_root}' ..."
  cd "${fake_root}"
  GOPATH="${TMP_GOPATH}" PATH="${TMP_GOPATH}/bin:${PATH}" hack/update-deps.sh

  # make sure the temp repo has no changes relative to the real repo
  echo "Diffing '${REPO_ROOT}/' '${fake_root}/' ..."
  diff=$(diff -ur \
          -x ".git" \
          -x "_output" \
         "${REPO_ROOT}" "${fake_root}" || true)
  if [[ -n "${diff}" ]]; then
    echo "unexpectedly dirty working directory after hack/update-deps.sh" >&2
    echo "${diff}" >&2
    echo "" >&2
    echo "please run: hack/update-deps.sh" >&2
    exit 1
  fi
}

main
