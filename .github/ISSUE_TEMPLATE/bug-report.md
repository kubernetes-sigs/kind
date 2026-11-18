---
name: Bug Report
about: Report a bug encountered using kind
labels: kind/bug

---

<!-- Please use this template while reporting a bug and provide as much info as possible. Not doing so may result in your bug not being addressed in a timely manner. Thanks!-->


**What happened**:

**What you expected to happen**:

**How to reproduce it (as _minimally_ and precisely as possible)**:

**Anything else we need to know?**:

**Cluster logs (if `kind create cluster` failed)**:

<!-- When `kubeadm init` / `kubeadm join` fail there are many possible causes, and the kubeadm output alone typically doesn't contain enough detail. The full cluster logs aid us greatly in diagnosing the failure. -->

1. Run the same create command again with `--retain`, so the nodes are not cleaned up on failure:
   `kind create cluster --retain <your other flags>`
2. Export the logs to a directory (add `--name <cluster-name>` if you used a non-default name):
   `kind export logs ./kind-logs`
3. Archive the directory: `tar -czf kind-logs.tar.gz ./kind-logs`
4. Attach `kind-logs.tar.gz` to this issue by dragging it into this text box (GitHub cannot upload a directory).
5. Clean up: `kind delete cluster`

**Environment:**

- kind version: (use `kind version`):
- Runtime info: (use `docker info`, `podman info` or `nerdctl info`):
- OS (e.g. from `/etc/os-release`):
- Kubernetes version: (use `kubectl version`):
- Any proxies or other special environment settings?:
