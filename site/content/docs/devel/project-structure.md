# Project Structure

## CLI
```
.
├── cmd
│   └── kind
│       ├── build/   # Build images
│       ├── create/  # Create cluster
│       ├── delete/  # Delete cluster
│       ├── get/     # List kubeconfigs and clusters
│       └── kind.go  # Root command
├── main.go          # Entrypoint
```

The CLI is built using [cobra][cobra] and you can see the app's entrypoint, [`main.go`][main.go], at the root level of the repository.
The CLI commands can be found in the directory [cmd][cmd]. Here, you will find
the root command [kind.go][kind.go] where we register other commands to build
images; create, delete, and list clusters; list kubeconfig files for cluster;
and setup logging.

## Packages
```
├── pkg
│   ├── build     # Build and manage images
│   ├── cluster   # Build and manage clusters
│   ├── docker    # Interact with Docker
│   ├── exec      # Execute commands
│   ├── fs        # Interact with the host file system
│   ├── kustomize # Work with embedded kustomize commands
│   ├── log       # Logging
│   └── util
```
`kind` commands rely on the functionality of the [packages directory][pkg].
Here, you will find everything needed to build container images for `kind`;
create clusters from these images; interact with the Docker engine and file system; customize configuration files; and logging.



[cobra]: https://github.com/spf13/cobra
[main.go]: ../../main.go
[cmd]: ../../cmd/kind/
[kind.go]: ../../cmd/kind/kind.go
[pkg]: ../../pkg
