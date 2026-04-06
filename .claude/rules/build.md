---
paths:
  - "Makefile"
  - "hack/build/**"
  - ".go-version"
  - ".tool-versions"
---

# Build Troubleshooting

## Skipping gimme (recommended when host Go matches `.go-version`)

The bundled gimme does not support Go versions newer than its own release. If your host Go
already matches the required version (e.g. via asdf), bypass gimme entirely with:

```bash
FORCE_HOST_GO=1 make clean build
make install INSTALL_DIR=$HOME/.local/bin
```

This avoids the warning:
```
Unable to setup go bootstrap from existing or binary
I don't have any idea what to do with '1.25.x'.
```

## Common errors

| Error | Cause | Fix |
|-------|-------|-----|
| `I don't have any idea what to do with '1.25.x'` | Bundled gimme is too old to handle this Go version | Use `FORCE_HOST_GO=1 make build` |
| `go: toolchain go1.25.x not available` | `GOTOOLCHAIN=local` is set in environment and host Go is older | Unset it: `unset GOTOOLCHAIN` |
| Binary still runs old version after install | Old binary earlier in `PATH` | Check with `which -a cloud-provisioner` |
| `go: updates to go.mod needed; to update it: go mod tidy` | `go.mod` is out of sync after a Go version bump | Run `go mod tidy` |
