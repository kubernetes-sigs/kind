module sigs.k8s.io/kind

// NOTE: This is the go language version, NOT the compiler version.
//
// This controls the *minimum* required go version and therefore available Go
// language features.
//
// See ./.go-version for the go compiler version used when building binaries
//
// https://go.dev/doc/modules/gomod-ref#go
go 1.17

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/alessio/shellescape v1.4.2
	github.com/evanphx/json-patch/v5 v5.6.0
	github.com/google/safetext v0.0.0-20220905092116-b49f7bc46da2
	github.com/mattn/go-isatty v0.0.20
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v3 v3.0.1
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
)
