# Versioning

`kind` follows [semver] with the following policies (possibly subject to change):

- The CLI will meet semver
- The configuration will meet semver along with Kubernetes API policies
- The go packages are best effort

Other than releases, versions will be `major.minor.patch-alpha`.

Maintainers will bump major / minor / patch versions between releases by running
`TYPE=minor hack/bump.sh` as necessary to bump and apply a git tag.

Releases will be uploaded to github, and be backed by default pre-built docker
images.


[semver]: https://semver.org/
