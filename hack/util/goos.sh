#!/bin/sh
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

# this script attempts to print GOOS for the host

# if we have go, just ask go!
if which go >/dev/null 2>&1; then
    go env GOOS
    exit $?
fi

# bash will set OSTYPE
if [ -n "${OSTYPE:-}" ]; then
  case "${OSTYPE}" in
    linux-gnu)
      echo "linux"
      exit 0
    ;;
    darwin)
      echo "darwin"
      exit 0
    ;;
  esac
fi

# fall back to uname
if which uname >/dev/null 2>&1; then
  case "$(uname -s)" in
    Darwin)
      echo "darwin"
      exit 0
    ;;
    Linux)
      echo "linux"
      exit 0
    ;;
  esac
fi

echo "Failed to detect a supported OS!"
exit 1