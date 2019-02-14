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
    # go module cache is not writable
    chmod -R 775 "${TMP_GOPATH}"
    rm -rf "${TMP_GOPATH}"
  fi
}

# cp -r without any warnings for symlinks ¯\_(ツ)_/¯
quiet_recursive_cp() {
  cp -r "${1}" "${2}" >/dev/null 2>&1
}

# copies repo into a temp root saved to TMP_GOPATH
make_temp_root() {
  # make a fake gopath
  local fake_root="${TMP_GOPATH}/src/sigs.k8s.io/kind"
  mkdir -p "${fake_root}"
  export -f quiet_recursive_cp
  # we need to copy everything but _output (which is .gitignore anyhow)
  find . \
    -mindepth 1 -maxdepth 1 \
    -type d -path "./_output" -prune -o \
    -exec bash -c 'quiet_recursive_cp "${0}" "${1}/${0}"' {} "${fake_root}" \;
}

main() {
  trap cleanup EXIT

  # copy repo root into tempdir under ./_output
  make_temp_root
  local fake_root="${TMP_GOPATH}/src/sigs.k8s.io/kind"

  # run deps update script
  cd "${fake_root}"
  GOPATH="${TMP_GOPATH}" PATH="${TMP_GOPATH}/bin:${PATH}" hack/update-deps.sh

  # make sure the temp repo has no changes relative to the real repo
  diff=$(diff -Nupr \
          -x ".git" \
          -x "_output" \
          -x "vendor/github.com/jteeuwen/go-bindata/testdata" \
          -x "vendor/github.com/golang/dep/internal/fs/testdata/symlinks" \
         "${REPO_ROOT}" "${fake_root}" 2>/dev/null || true)
  if [[ -n "${diff}" ]]; then
    echo "unexpectedly dirty working directory after hack/update-deps.sh" >&2
    echo "" >&2
    echo "${diff}" >&2
    echo "" >&2
    echo "please run hack/update-deps.sh" >&2
    exit 1
  fi
}

main
