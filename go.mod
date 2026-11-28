module sigs.k8s.io/kind

// NOTE: This is the go language version, NOT the compiler version.
//
// This controls the *minimum* required go version and therefore available Go
// language features.
//
// See ./.go-version for the go compiler version used when building binaries
//
// https://go.dev/doc/modules/gomod-ref#go
go 1.21

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/BurntSushi/toml v1.6.0
	github.com/evanphx/json-patch/v5 v5.9.11
	github.com/mattn/go-isatty v0.0.24
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	go.yaml.in/yaml/v3 v3.0.5
	sigs.k8s.io/yaml v1.4.0
)

// test-only transitive deps, these are used by sigs.k8s.io/yaml's tests
require gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)
