#!/usr/bin/env bash

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

# CI script to run shellcheck
set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# required version for this script, if not installed on the host we will
# use the official docker image instead. keep this in sync with SHELLCHECK_IMAGE
SHELLCHECK_VERSION="0.9.0"
# upstream shellcheck latest stable image as of October 23rd, 2019
SHELLCHECK_IMAGE='docker.io/koalaman/shellcheck-alpine:v0.9.0@sha256:e19ed93c22423970d56568e171b4512c9244fc75dd9114045016b4a0073ac4b7'

DOCKER="${DOCKER:-docker}"

# detect if the host machine has the required shellcheck version installed
# if so, we will use that instead.
HAVE_SHELLCHECK=false
if which shellcheck &>/dev/null; then
  detected_version="$(shellcheck --version | grep 'version: .*')"
  if [[ "${detected_version}" = "version: ${SHELLCHECK_VERSION}" ]]; then
    HAVE_SHELLCHECK=true
  fi
fi

# Find all shell scripts excluding:
# - Anything git-ignored - No need to lint untracked files.
# - ./.git/* - Ignore anything in the git object store.
# - ./vendor/* - Ignore vendored contents.
# - ./bin/* - No need to lint output directories.
all_shell_scripts=()
while IFS=$'\n' read -r script;
  do git check-ignore -q "$script" || all_shell_scripts+=("$script");
done < <(grep -irl '#!.*sh' . | grep -Ev '(^\./\.git/)|(^\./vendor/)|(^\./hack/third_party/)|(^\./images/base/scripts/third_party/)|(^\./bin/)|(\.go$)')

# common arguments we'll pass to shellcheck
SHELLCHECK_OPTIONS=(
  # allow following sourced files that are not specified in the command,
  # we need this because we specify one file at a time in order to trivially
  # detect which files are failing
  '--external-sources'
  # disabled lint codes
  # 2330 - disabled due to https://github.com/koalaman/shellcheck/issues/1162
  '--exclude=2230'
  # 2126 - disabled because grep -c exits error when there are zero matches,
  # unlike grep | wc -l
  '--exclude=2126'
  # set colorized output
  '--color=auto'
)

# actually shellcheck
# tell the user which we've selected and lint all scripts
# The shellcheck errors are printed to stdout by default, hence they need to be redirected
# to stderr in order to be well parsed for Junit representation by juLog function
res=0
if ${HAVE_SHELLCHECK}; then
  >&2 echo "Using host shellcheck ${SHELLCHECK_VERSION} binary."
  shellcheck "${SHELLCHECK_OPTIONS[@]}" "${all_shell_scripts[@]}" >&2 || res=$?
else
  >&2 echo "Using shellcheck ${SHELLCHECK_VERSION} docker image."
  "${DOCKER}" run \
    --rm -v "${REPO_ROOT}:${REPO_ROOT}" -w "${REPO_ROOT}" \
    "${SHELLCHECK_IMAGE}" \
  shellcheck "${SHELLCHECK_OPTIONS[@]}" "${all_shell_scripts[@]}" >&2 || res=$?
fi
exit $res
