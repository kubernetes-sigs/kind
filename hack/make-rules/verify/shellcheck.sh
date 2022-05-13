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

# upstream shellcheck latest stable image as of January 10th, 2019
SHELLCHECK_IMAGE='koalaman/shellcheck:v0.7.1'

# Find all shell scripts excluding:
# - Anything git-ignored - No need to lint untracked files.
# - ./.git/* - Ignore anything in the git object store.
# - ./vendor/* - Ignore vendored contents.
# - ./bin/* - No need to lint output directories.
all_shell_scripts=()
while IFS=$'\n' read -r script;
  do git check-ignore -q "$script" || all_shell_scripts+=("$script");
done < <(grep -irl '#!.*sh' . | grep -Ev '(^\./\.git/)|(^\./vendor/)|(^\./bin/)|(\.go$)')

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
docker run \
  --rm -t -v "${REPO_ROOT}:${REPO_ROOT}" -w "${REPO_ROOT}" \
  "${SHELLCHECK_IMAGE}" \
  "${SHELLCHECK_OPTIONS[@]}" "${all_shell_scripts[@]}"
