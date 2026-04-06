# Go Code Writer — Agent Memory

## Build Environment Note
Makefile uses `GOTOOLCHAIN=auto`. If host Go matches `.go-version`, use `FORCE_HOST_GO=1 make build` to skip gimme.
Tests run fine with host Go: `go test ./pkg/commons/... -v -run TestName`.

## Key Patterns

### CentralECR Registry Prefixing
`commons.GetPrefixedRegistryURL(originalRegistry, baseURL, awsCentralECREnabled)` in `pkg/commons/cluster.go`.
- docker.io → /dockerhub, public.ecr.aws → /ecrpublic, ghcr.io → /ghcr, quay.io → /quay, k8s.io → /k8s
- Uses `strings.Contains` so "registry.k8s.io" also matches the k8s.io case.
- Returns baseURL unchanged when disabled or baseURL is empty.

### installCalico (provider.go) Pattern
`calicoHelmParams` struct has both `KeosRegUrl` (docker.io images) and `QuayRegUrl` (quay.io/tigera).
Both are set at struct init time via `GetPrefixedRegistryURL`, NOT via a conditional block after init.
Template `tigera-operator-helm-values.tmpl`: `installation.registry` uses `$.KeosRegUrl`,
`tigeraOperator.registry` uses `$.QuayRegUrl`.

### PrivateParams Copy Pattern (createworker.go)
When a function needs a modified KeosRegUrl for a sub-call without mutating the caller's copy:
```go
localParams := privateParams
localParams.KeosRegUrl = commons.GetPrefixedRegistryURL("registry.k8s.io", privateParams.KeosRegUrl, privateParams.CentralECR)
result, err := getManifest(..., localParams)
```

### Test Conventions
- Package: same package (not `_test` suffix) for whitebox tests.
- Table-driven with `t.Run`. File name: `<source_file>_test.go` in same directory.
- Run: `go test ./pkg/commons/... -v -run TestFunctionName`
