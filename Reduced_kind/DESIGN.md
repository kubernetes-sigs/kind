# Multi-host Reduced_kind via Docker Swarm

> Status: design / thinking only.  No code yet.

## TL;DR

Use Docker Swarm **only** for its overlay network.  Use Docker contexts to
target each host.  The kind code-base barely changes because the Swarm
overlay makes "many hosts" look like "one big virtual host" to everything
above the network layer.

```
docker swarm init                                 (host A)
docker swarm join --token X mgr-A                 (host B, C)

docker context create host-a   --docker host=ssh://user@A
docker context create host-b   --docker host=ssh://user@B
docker context create host-c   --docker host=ssh://user@C

docker --context=host-a network create -d overlay --attachable kind

docker --context=host-a run --network=kind --name=foo-control-plane ...
docker --context=host-b run --network=kind --name=foo-worker        ...
docker --context=host-c run --network=kind --name=foo-worker2       ...
```

That's the whole technique.  Everything else is plumbing.

---

## Why this works (and why it's almost free)

kind's existing abstractions assume *one* docker daemon and *one* network.
Either of those, naively, would prevent multi-host.  Swarm + contexts neutralise
both:

1. **Cross-host networking:** Swarm's `overlay` driver spans every host that
   is a Swarm member.  Containers attached to it can reach each other by name
   regardless of which host they actually run on.  Existing kindnetd sees them
   as a single L2 segment and its current "add a route per pod CIDR" loop
   keeps working unchanged.

2. **Cross-host docker control:** A docker context is a connection profile.
   `docker --context=X run …` is exactly `docker run …` against host X's
   daemon.  No new SDK, no SSH wrappers; the kind code's habit of building
   `exec.Command("docker", …)` extends with a single `--context=X` argument.

3. **Service discovery:** Swarm's embedded DNS resolves `<container-name>` to
   the right overlay IP across all hosts.  kind already relies on this for
   the *single* host case (`<cluster>-control-plane:6443`).  No code change.

What we **don't** use Swarm for:

- `docker service` / replicas / scheduling.  Kind nodes are pet-like; we want
  fixed names, fixed mounts, deterministic placement.  We `docker run` plain
  containers and only borrow the overlay network plus DNS.

---

## Architectural change vs kind

```
┌─ kind ────────────────────┐    ┌─ Reduced_kind multi-host ─────────────┐
│                           │    │                                       │
│  Host A                   │    │  Host A             Host B            │
│  ┌─────────────────────┐  │    │  ┌───────────────┐  ┌───────────────┐ │
│  │ docker bridge "kind"│  │    │  │ daemon A      │  │ daemon B      │ │
│  │  ┌─┐ ┌─┐ ┌─┐        │  │    │  │ ┌─┐           │  │ ┌─┐ ┌─┐       │ │
│  │  │c│ │w│ │w│        │  │    │  │ │c│           │  │ │w│ │w│       │ │
│  │  └─┘ └─┘ └─┘        │  │    │  │ └┬┘           │  │ └┬┘ └┬┘       │ │
│  └─────────────────────┘  │    │  └──┼────────────┘  └──┼───┼────────┘ │
│                           │    │     │  Swarm overlay   │   │          │
│                           │    │     └────"kind"────────┴───┘          │
└───────────────────────────┘    └───────────────────────────────────────┘
```

The control plane container (`c`) on host A and worker containers (`w`) on
host B share **one logical network**.  All node-to-node traffic crosses an
encapsulated VXLAN tunnel that Swarm sets up between daemons.

---

## Code-level changes (still small)

### Config schema (already done in `config/types.go`)

- `Node.Host` — which docker context to use.  Empty = local context.
- `Networking.SwarmOverlay` — if any `Node.Host` is set this is auto-true.

### Provider changes

Reuse the kind docker provider almost verbatim.  Three surgical changes:

1. **`exec.Command("docker", …)` → `dockerCmd(host, …)`** where
   `dockerCmd("", "ps")` → `docker ps` and `dockerCmd("host-b", "ps")` →
   `docker --context=host-b ps`.  Single helper, threaded through provider
   methods (`ListClusters`, `ListNodes`, `provision.go`, etc.).

2. **Network creation**

   ```go
   // single-host
   docker network create -d bridge --label … kind

   // multi-host (Reduced_kind)
   docker --context=<any swarm mgr> network create \
     -d overlay --attachable --label … kind
   ```

   Choose any Swarm manager context.  All workers see the network because
   Swarm gossips the network spec.

3. **`ListNodes` / `ListClusters` aggregation**

   Containers attached to an overlay are *not* tracked by Swarm itself; only
   their DNS entries are.  So `docker --context=host-X ps` only sees host X's
   containers.  We must fan out across every known host context, dedupe by
   container ID, and merge.  Two extra system calls per cluster operation;
   negligible.

### Node interface — already correct

`pkg/cluster/nodes/Node` is an interface; the docker implementation simply
prepends `--context=<host>` to its emitted commands.  Every consumer
(`actions/`, `kubeconfig/`, `logs/`) keeps working.

### Actions — no changes

All seven actions in `pkg/cluster/internal/create/actions/` operate through
the `Node` interface.  Once the underlying `node.Command(...)` correctly
addresses the right docker daemon, every action — including `kubeadm init`,
certificate copy, kubeadm join — works unchanged.

### `kind load image` — minor change

Reduced_kind's analogue must `docker save | docker --context=X load` against
**every** host that hosts a node.  Or, much simpler, point all hosts at a
shared registry and require the user to push there.

---

## What still needs thinking

### Image distribution

Each host needs `kindest/node:vX.Y.Z` locally before `docker run`.
Three options:

| option | pros | cons |
| ------ | ---- | ---- |
| Pull on each host on demand | zero infra | slow for big images, repeated work |
| Shared registry (mirror) | fast, deduped | user must run a registry |
| `docker save \| docker load` over context | self-contained | each `kind create` re-copies |

Recommend: pull on demand (the kind UX), document a registry-mirror as the
"go faster" knob.

### API server endpoint visible to the user

In single-host kind, the user's `kubectl` connects via
`127.0.0.1:<random-host-port>` mapped from the control-plane container.

Multi-host:

- The control plane runs on a specific host (say `host-a`).
- Publish `6443` on `host-a` to a known port.
- `kubeconfig.server` is `https://<host-a's reachable IP>:<port>`.
- If the user's `kubectl` is on `host-a` itself: same as kind today.
- If `kubectl` lives elsewhere (laptop): user must use a routable IP.
  We can ask the user to specify which host's external IP to advertise via
  `--api-server-host` flag.

### HAProxy LB for multi-CP

Already covered by the overlay: drop the LB container on any swarm node and
publish 6443 on that host's external interface.  Kind's existing
`loadbalancer/` action and Envoy config work because they refer to backends
by container name (Swarm DNS resolves them).

### `extraMounts.hostPath`

Path is host-specific.  Reduced_kind must require absolute paths (kind
already does) and document that the path must exist on the **target host**,
not the user's workstation.

### MTU

Swarm overlay adds ~50 bytes of VXLAN header.  Default MTU of 1500 minus 50
gives 1450.  Some environments (cloud VPCs with jumbo frames disabled) need
explicit MTU on the overlay.  Add a `Networking.OverlayMTU` knob, default
1450.

### Tearing down

Cleanup must:

1. Delete every node container on its respective host
2. Delete the overlay network (only after every container is gone — Swarm
   refuses otherwise)
3. **Not** leave the Swarm itself.  Users may have Swarm for other reasons.

### Failure modes we accept

- Host loses connectivity: the node container is gone for the rest of the
  cluster's lifetime.  We don't reschedule (kind nodes are stateful).
- Host reboots: `docker run --restart=on-failure:1` brings the container
  back, exactly like single-host kind.
- Swarm manager dies: existing containers keep running, but `docker network
  create/delete` for the overlay fails.  Document quorum requirements.

---

## What we explicitly drop from kind

To stay "Reduced", these features go:

- **Podman / Nerdctl providers.**  The whole multi-host story is bound to
  `docker` + Swarm.  Re-introducing alternatives doubles complexity.
- **Rootless detection.**  Swarm + rootless is fragile; require rootful.
- **userns-remap detection.**
- **Btrfs / ZFS `/dev/mapper` hack.**
- **IPv6 subnet retry loops in network creation.**  Swarm's IPAM handles
  it; if it fails we fail loudly and tell the user.
- **`kind build node-image`.**  Consume upstream images.

---

## Phased implementation plan

When (if) we eventually code this:

1. **Phase 1 — single host, contexts only.**  Implement `--context` plumbing
   and prove kind still works with `docker --context=default`.  Zero behaviour
   change but flushes out the new code paths.

2. **Phase 2 — multi-host on Swarm.**  Add overlay network, per-node host
   targeting, fan-out `ListNodes`.  Tested with a 2-host cluster on local VMs.

3. **Phase 3 — quality of life.**  `--api-server-host` flag, `extraMounts`
   target validation, image-distribution helper for `kind load image`.

---

## Open questions

- Do we want Reduced_kind to manage Swarm itself (`docker swarm init` if not
  initialised), or require the user to set it up?  Leaning toward "require"
  to keep the tool focused.
- Should the overlay be `--encrypted`?  Default is plaintext VXLAN; encryption
  adds CPU cost.  Probably opt-in via config.
- How do we handle a user who specifies `Host: x` but `x` isn't a swarm
  member?  Validate eagerly at cluster create time.
