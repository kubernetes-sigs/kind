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

# inputs are:
# - LAST_VERSION_TAG -- This is the version to get commits since
#    like: LAST_VERSION_TAG="v0.8.1"
# - GITHUB_OATH_TOKEN -- used to avoid hitting API rate limits
ORG="kubernetes-sigs"
REPO="kind"

# query git for contributors since the tag
contributors=()
while IFS='' read -r line; do contributors+=("$line"); done < <(git log --format="%aN <%aE>" "${LAST_VERSION_TAG:?}.." | sort | uniq)

# query github for usernames and output bulleted list
contributor_logins=()
for contributor in "${contributors[@]}"; do
    # get a commit for this author
    commit_for_contributor="$(git log --author="${contributor}" --pretty=format:"%H" -1)"
    # lookup the  commit info to get the login
    contributor_logins+=("$(curl \
        -sG \
        ${GITHUB_OAUTH_TOKEN:+-H="Authorization: token ${GITHUB_OAUTH_TOKEN:?}"} \
        --data-urlencode "q=${contributor}" \
        "https://api.github.com/repos/${ORG}/${REPO}/commits/${commit_for_contributor}" \
    | jq -r .author.login
    )")
done

echo "Contributors since ${LAST_VERSION_TAG}:"
# echo sorted formatted list
while IFS='' read -r contributor_login; do
     echo "- @${contributor_login}"
done < <(for c in "${contributor_logins[@]}"; do echo "$c"; done | LC_COLLATE=C sort --ignore-case | uniq)
