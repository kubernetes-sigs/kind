#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
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

# Requires git, gh, and jq. Run from the local repository being queried.
# Authenticate with gh auth login, GH_TOKEN, or GITHUB_OAUTH_TOKEN.
# Usage: get-contributors.sh [REVISION_RANGE]
# For example: get-contributors.sh v0.30.0..upstream/main
# Defaults to HEAD, including all history reachable from it.
# Optional environment variables:
# - ORG / REPO -- GitHub repository (defaults to kubernetes-sigs/kind)
set -o errexit
set -o nounset
set -o pipefail

if [[ "$#" -gt 1 ]]; then
    echo "Usage: $0 [REVISION_RANGE]" >&2
    exit 1
fi
revision="${1:-HEAD}"

ORG="${ORG:-kubernetes-sigs}"
REPO="${REPO:-kind}"
if [[ -n "${GITHUB_OAUTH_TOKEN:-}" ]]; then
    export GH_TOKEN="${GITHUB_OAUTH_TOKEN}"
fi

commits="$(git log --format='%H%x09%aN <%aE>%n%(trailers:key=Co-authored-by,valueonly)' "${revision}" -- | awk '
    index($0, "\t") {
        commit = substr($0, 1, index($0, "\t") - 1)
        $0 = substr($0, index($0, "\t") + 1)
    }
    NF && !authors[$0]++ && !commits[commit]++ { print commit }
')"

query="$(cat <<'GRAPHQL'
query($owner: String!, $name: String!, $oid: GitObjectID!, $endCursor: String) {
    repository(owner: $owner, name: $name) {
        object(oid: $oid) {
            ... on Commit {
                authors(first: 100, after: $endCursor) {
                    nodes { name email user { login } }
                    pageInfo { hasNextPage endCursor }
                }
            }
        }
    }
}
GRAPHQL
)"

output_dir="$(mktemp -d)"
trap 'rm -rf "${output_dir}"' EXIT
touch "${output_dir}/logins"
while IFS= read -r commit; do
    [[ -n "${commit}" ]] || continue
    response="$(gh api graphql --paginate \
        -f query="${query}" -f owner="${ORG}" -f name="${REPO}" -f oid="${commit}")"
    if ! jq -e -s 'length > 0 and all(.[];
        (.errors | length) == 0 and
        (.data.repository.object.authors.nodes | type) == "array")' \
        <<< "${response}" > /dev/null; then
        echo "Invalid GitHub author response for ${commit}" >&2
        exit 1
    fi
    jq -r '.data.repository.object.authors.nodes[] |
        select(.user.login == null or .user.login == "") |
        "No GitHub account for \(.name) <\(.email)>"' <<< "${response}" >&2
    jq -r '.data.repository.object.authors.nodes[] | .user.login // empty |
        select(length > 0)' <<< "${response}" >> "${output_dir}/logins"
done <<< "${commits}"

echo "Contributors in ${revision}:"
LC_ALL=C sort -f -u "${output_dir}/logins" | while IFS= read -r login; do
    echo "- @${login}"
done
