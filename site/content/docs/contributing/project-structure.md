---
title: "Project Structure"
menu:
  main:
    parent: "contributing"
    identifier: "project-structure"
---
# Project Structure

🚧 This is a work-in-progress 🚧

## Code Layout
The kind project is composed of two parts: the CLI and the packages that        
implement kind's functionality.
We will go more in depth below.

### CLI

kind's CLI commands are defined in [cmd/kind][cmd].
Each subdirectory corresponds to a command, and each of their subdirectories
implements a subcommand.
The CLI is built using [cobra][cobra] and you can see the app's entrypoint,

### Packages
```
├── pkg
│   ├── build      # Build and manage images
│   ├── cluster    # Build and manage clusters
│   ├── concurrent # Utilities for running functions concurrently
│   ├── container  # Interact with the host's container runtime
│   ├── exec       # Execute commands
│   ├── fs         # Interact with the host file system
│   ├── kustomize  # Work with embedded kustomize commands
│   ├── log        # Logging
│   └── util
```
`kind` commands rely on the functionality of the [packages directory][pkg].
Here, you will find everything needed to build container images for `kind`;
create clusters from these images; interact with the Docker engine and file system; customize configuration files; and logging.


## Developer Tooling
Kind includes some tools to help developers maintain the source code compliant to Go best coding practices using tools such as Go fmt, Go lint, and Go vet. It also includes utility scripts that will generate code necessary for kind to make use of Kubernetes-style resource definitions.

Tools are included in the [hack/][hack] directory and fall in one of two categories:

* `update-*` : make changes to the source code
* `verify-*` : verify that source code is in a good state

We will proceed by describing all of the current tooling in [hack/][hack].

### Verify
You can check the compliance of the entire project by running the `verify-all.sh` script. This script will do the following:

* check spelling using [client9/misspell](https://github.com/client9/misspell)
* check that the code is properly formatted using Go fmt
* check that all source code (except vendored and generated code) successfully passes Go lint’s style rules
* verify that all code successfully passes Go vet’s tests
* verify that any of the generated files is up to date
* verify vendored dependencies are present


### Update
In order to get the project’s source code into a compilable state, the script
**[update-all.sh](https://sigs.k8s.io/kind/hack/update/all.sh)** performs the following tasks:

* runs update-deps.sh to obtain all of the project’s dependencies
* runs update-generated.sh to generate code necessary to generate Kubernetes API code for kind, see https://kind.sigs.k8s.io/docs/user/quick-start/#configuring-your-kind-cluster
* runs update-gofmt.sh which formats all Go source code using the Go fmt tool.

Let’s go a little in depth on each of these files.

**[update-deps.sh](https://sigs.k8s.io/kind/hack/update/deps.sh)** performs the following steps:

* runs `go mod tidy` to remove any no-longer-needed dependencies from go.mod and add any dependencies needed for other combinations of OS, architecture, and build tags
* runs `go mod vendor`, which re-populates the vendor directory, resets the main module's vendor directory to include all packages needed to build and test all of the module's packages based on the state of the go.mod files and Go source code.
* finally, it prunes the vendor directory of any unnecessary files (keeps code source files, licenses, author files, etc).

**[update-gofmt.sh](https://sigs.k8s.io/kind/hack/update/gofmt.sh)** runs gofmt on all nonvendored source code to format and simplify the code.

**[update-generated.sh](https://sigs.k8s.io/kind/hack/update/generated.sh)** in short, generates Go source code that is necessary to use a Kubernetes-style resource definition to define a schema that can be use to configure Kind.

Going a bit more in depth, update-generated.sh does the following:

* Installs [go-bindata](https://github.com/jteeuwen/go-bindata) which is used to embed static assets in Go.
* Installs the deepcopy-gen, defaulter-gen, and conversion-gen tools from [kubernetes/code-generator](https://github.com/kubernetes/code-generator).

These programs are used to generate Kubernetes-like APIs. These programs are run in the following sequence: deepcopy -> defaulter -> conversion.

To understand this process better, we need to keep in mind that kind’s configuration schema dictates how to bootstrap a Kubernetes cluster. The schema is defined in
[kind/pkg/cluster/config](https://sigs.k8s.io/kind/pkg/cluster/config).
In this directory, currently, you will see two subdirectories:
[v1alpha3][v1alpha3].
Each of these subdirectories corresponds to a version of kind’s cluster configuration.

One of the concerns with versioned configurations is enabling the project to be compatible with old schema versions.
With this in mind, [kind/pkg/cluster/config](https://sigs.k8s.io/kind/pkg/cluster/config) contains the internal configuration fields
which are used as the basis for conversion between the external types
(i.e. [v1alpha3][v1alpha3]).

The way this is implemented is by running deepcopy-gen and defaulter-gen in
[kind/pkg/cluster/config](https://sigs.k8s.io/kind/pkg/cluster/config),
followed by running deepcopy-gen, defaulter-gen, and conversion-gen on all version subdirectories
(i.e. [v1alpha3][v1alpha3]).

The [kubernetes/code-generator](https://github.com/kubernetes/code-generator) tools work by comment tags which are specified in the `doc.go` file for each directory. For example, all `doc.go` files within [kind/pkg/cluster/config](https://sigs.k8s.io/kind/pkg/cluster/config) have the following tags:
```
// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=sigs.k8s.io/kind/pkg/cluster/config
// +k8s:defaulter-gen=TypeMeta
```

Additionally, [pkg/cluster/config/types.go](https://sigs.k8s.io/kind/pkg/cluster/config/types.go) has an additional tag:
```
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
```

Let’s see how this tags work with the
[kubernetes/code-generator](https://github.com/kubernetes/code-generator)
binaries deepcopy-gen, defaulter-gen, and conversion-gen.

For each of the directories related to defining a configuration for kind, we start by running [deepcopy-gen](https://godoc.org/k8s.io/code-generator/cmd/deepcopy-gen). deepcopy-gen generates functions that efficiently perform a full deep-copy of each type that is part of the configuration.

Once we have these utility functions in place then we will need to run
[defaulter-gen](https://godoc.org/k8s.io/code-generator/cmd/defaulter-gen)
to generate efficient defaulters (functions that will fill in default value for configuration fields) for the configuration schema based on the
[`Config`](https://sigs.k8s.io/kind/pkg/cluster/config/types.go)
and the [`Node`](https://sigs.k8s.io/kind/pkg/cluster/config/types.go) types.
The way this works is that the
`// +k8s:defaulter-gen=TypeMeta`
comment tag will generate a defaulter for the `Config` type as this possesses a `TypeMeta` field.

The `TypeMeta` field is defined in
[k8s.io/apimachinery/pkg/apis/meta/v1](https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#TypeMeta).
`TypeMeta` is a struct with a Kind and APIVersion fields. Structures that are versioned or persisted should inline TypeMeta.


The final step in code generation for kind’s configuration specification involves running
[conversion-gen](https://godoc.org/k8s.io/code-generator/cmd/conversion-gen)
which will scan its input directories, looking at the package defined in each of those directories for comment tags that define a conversion code generation task. In Kind, you will see the following comment tag
```
// +k8s:conversion-gen=sigs.k8s.io/kind/pkg/cluster/config
```
which introduces a conversion task for which the destination package (the top level configuration definition in [kind/pkg/cluster/config](https://sigs.k8s.io/kind/pkg/cluster/config)) is the one containing the file with the tag.
This last step builds on the deep copy generators and defaulters previously created to enable kind to understand any known configuration version.




[cobra]: https://github.com/spf13/cobra
[cmd]: https://sigs.k8s.io/kind/cmd/kind/
[hack]: https://sigs.k8s.io/kind/hack/
[kind.go]: https://sigs.k8s.io/kind/cmd/kind/kind.go
[pkg]: https://sigs.k8s.io/kind/pkg
[v1alpha3]: https://sigs.k8s.io/kind/pkg/cluster/config/v1alpha3
