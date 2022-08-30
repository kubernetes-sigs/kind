// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
module github.com/verrazzano/kind

go 1.14

replace sigs.k8s.io/kind => ./

require (
	github.com/BurntSushi/toml v1.0.0
	github.com/alessio/shellescape v1.4.1
	github.com/evanphx/json-patch/v5 v5.6.0
	github.com/mattn/go-isatty v0.0.14
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v3 v3.0.1
	sigs.k8s.io/kind v0.14.0
	sigs.k8s.io/yaml v1.3.0
)
