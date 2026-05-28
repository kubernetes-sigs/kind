# kind Release Process

This document describes the steps to cut a new kind release. It is intended for
maintainers who have push access to the upstream repository and the staging
image registry.

## Prerequisites

- GNU sed (macOS: `brew install gnu-sed`)
- Docker with buildx support
- [`crane`](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md) installed (for image promotion to Docker Hub)
- Push access to `github.com/kubernetes-sigs/kind`
- Push access to `gcr.io/k8s-staging-kind`
- Push access to `kindest` on Docker Hub

---

## Phase 1 - Publish Node Images

Node images must be built, tested, and promoted to Docker Hub **before** the
kind release so their digest hashes are known and can be embedded in the
release binary.

### 1.1 Build and push to staging

Use `hack/release/push-node.sh` for Kubernetes v1.31 and later:

```bash
./hack/release/push-node.sh v1.35.0
```

This builds `amd64` and `arm64` node images and pushes them to
`gcr.io/k8s-staging-kind/node:v1.35.0`, then creates a multi-arch manifest
list.

You can override the registry or add architectures:

```bash
REGISTRY=gcr.io/k8s-staging-kind ARCHES="amd64 arm64" ./hack/release/push-node.sh v1.35.0
```

### 1.2 Test with the staging image

Update the default image in `pkg/apis/config/defaults/image.go` to point to the
staging image with its digest, then run CI to validate:

```go
const Image = "gcr.io/k8s-staging-kind/node:v1.35.0@sha256:<digest-from-push>"
```

Get the digest from the push output, or by inspecting the manifest:

```bash
crane digest gcr.io/k8s-staging-kind/node:v1.35.0
```

### 1.3 Promote the image to Docker Hub

Once testing passes, copy the image from staging to `kindest/node` on Docker Hub:

```bash
crane cp \
  gcr.io/k8s-staging-kind/node:v1.35.0@sha256:<staging-digest> \
  kindest/node:v1.35.0
```

After promotion, retrieve the Docker Hub digest (it may differ from the staging
digest):

```bash
crane digest kindest/node:v1.35.0
```

### 1.4 Update the default image to the promoted version

Update `pkg/apis/config/defaults/image.go` with the promoted `kindest/node`
reference and its digest, then open a PR:

```go
const Image = "kindest/node:v1.35.0@sha256:<dockerhub-digest>"
```

---

## Phase 2 - Cut the kind Release

### 2.1 Ensure the tree is ready

- The default node image (`pkg/apis/config/defaults/image.go`) should reference
  a promoted `kindest/node` image, not a staging image.
- All CI is green on `main`.
- The current alpha version in `pkg/cmd/kind/version/version.go` matches the
  version you are about to release (e.g., `versionCore = "0.31.0"` with
  `versionPreRelease = "alpha"`).

### 2.2 Run the release script

```bash
./hack/release/create.sh 0.31.0 0.32.0
```

The two arguments are:
- `0.31.0` - the version being released
- `0.32.0` - the next version (the script will create a `0.32.0-alpha` pre-release)

The script will:

1. Set `versionCore = "0.31.0"` and `versionPreRelease = ""` in
   `pkg/cmd/kind/version/version.go`
2. Commit with message `version v0.31.0` and tag `v0.31.0`
3. Build cross-platform release binaries into `bin/`:
   - `kind-linux-amd64`, `kind-linux-arm64`
   - `kind-darwin-amd64`, `kind-darwin-arm64`
   - `kind-windows-amd64`
   - `.sha256sum` file for each binary
4. Set `versionCore = "0.32.0"` and `versionPreRelease = "alpha"`
5. Commit with message `version v0.32.0-alpha` and tag `v0.32.0-alpha`

At the end the script prints follow-up instructions. Continue with the steps
below.

### 2.3 Push the commits as a PR

```bash
git push origin HEAD
```

Open a PR with the two version commits against `main` and wait for it to merge.

### 2.4 Push the tags to upstream

After the PR is merged, push both tags to the upstream repository:

```bash
git push https://github.com/kubernetes-sigs/kind.git v0.31.0
git push https://github.com/kubernetes-sigs/kind.git v0.32.0-alpha
```

### 2.5 Generate the contributor list

While the PR is merging or shortly after, generate the list of contributors since
the previous release to include in the GitHub release notes:

```bash
LAST_VERSION_TAG=v0.30.0 GITHUB_OAUTH_TOKEN=<token> ./hack/release/get-contributors.sh
```

The token is optional but avoids GitHub API rate limits. The output is a
markdown-formatted bulleted list of GitHub usernames.

### 2.6 Create the GitHub release

Go to https://github.com/kubernetes-sigs/kind/releases/new and:

1. Select the tag `v0.31.0`.
2. Set the title to `v0.31.0`.
3. Write release notes summarizing changes since the previous release. Include
   the contributor list generated above.
4. Upload the binaries from `bin/` as release assets:
   - `kind-linux-amd64` + `kind-linux-amd64.sha256sum`
   - `kind-linux-arm64` + `kind-linux-arm64.sha256sum`
   - `kind-darwin-amd64` + `kind-darwin-amd64.sha256sum`
   - `kind-darwin-arm64` + `kind-darwin-arm64.sha256sum`
   - `kind-windows-amd64` + `kind-windows-amd64.sha256sum`
5. Publish the release.

---

## Phase 3 - Update Documentation

After the GitHub release is published, update the documentation to reference
the new stable version. This is typically done as a separate PR immediately
after the release.

### 3.1 Update README.md

Replace all occurrences of the old version string (e.g., `v0.30.0`) with the
new one (`v0.31.0`). The README contains several hardcoded download URLs and
`go install` commands that need updating.

### 3.2 Update site/config.toml

Update the `stable` parameter:

```toml
[params]
stable = "v0.31.0"
```

This value is used by the `{{< stableVersion >}}` shortcode throughout the
website documentation, so this single change updates all version references on
the site.

### 3.3 Open a PR

Commit both files and open a PR. The commit message convention is:

```
bump docs to 0.31.0
```

---

## Summary Checklist

### Node images (before kind release)

- [ ] Build and push node image(s) to staging: `./hack/release/push-node.sh vX.Y.Z`
- [ ] Test with staging image in `pkg/apis/config/defaults/image.go`
- [ ] Promote to Docker Hub: `crane cp gcr.io/k8s-staging-kind/node:vX.Y.Z@sha256:... kindest/node:vX.Y.Z`
- [ ] Update `pkg/apis/config/defaults/image.go` to promoted `kindest/node` reference + merge PR

### kind release

- [ ] Confirm `main` is green and default image is the promoted `kindest/node` image
- [ ] Run `./hack/release/create.sh <version> <next-version>`
- [ ] Open PR with the two version commits, wait for merge
- [ ] Push tags: `git push upstream v<version>` and `git push upstream v<next-version>-alpha`
- [ ] Generate contributor list: `LAST_VERSION_TAG=v<prev-version> ./hack/release/get-contributors.sh`
- [ ] Create GitHub release from `v<version>` tag, upload binaries from `bin/`

### Documentation

- [ ] Update `README.md` - replace all old version references
- [ ] Update `site/config.toml` - set `stable = "v<version>"`
- [ ] Open docs PR and merge
