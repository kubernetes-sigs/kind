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

# this script attempts to print GOARCH for the host, even if go is not installed

# if we have go, just ask go!
if which go >/dev/null 2>&1; then
    go env GOARCH
    exit $?
fi

# fall back to uname
if which uname >/dev/null 2>&1; then
  case "$(uname -m)" in
    x86_64)
      echo "amd64"
      exit 0
    ;;
    arm*)
      if [ "$(getconf LONG_BIT)" = "64" ]; then
        echo "arm64"
        exit 0
      else
        echo "arm"
        exit 0
      fi
    ;;
  esac
fi

echo "Failed to detect a supported OS!"
exit 1