---
name: Bug Report
about: Report a bug encountered using kind
labels: kind/bug

---

<!-- Please use this template while reporting a bug and provide as much info as possible. Not doing so may result in your bug not being addressed in a timely manner. Thanks!-->


**What happened**:

<!-- If creating a cluster failed, please run the create cluster command again with the `--retain` flag to prevent cleanup on failure, then run `kind export logs` to dump the cluster logs before cleaning up manually with `kind delete cluster`. Then attach the logs from the path printed by `kind export logs` to this issue as a zip or tarball archive. This will aid us greatly in diagnosing the failure. When `kubeadm init` / `kubeadm join` fail, there are many possible cases and the kubeadm logs typically don't contain enough details vs the full cluster logs. Thanks!-->

**What you expected to happen**:

**How to reproduce it (as _minimally_ and precisely as possible)**:

**Anything else we need to know?**:

**Environment:**

- kind version: (use `kind version`):
- Runtime info: (use `docker info` or `podman info`):
- OS (e.g. from `/etc/os-release`):
- Kubernetes version: (use `kubectl version`):
- Any proxies or other special environment settings?:
