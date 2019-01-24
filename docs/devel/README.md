# Get the required software for Linux and MacOS
Here we will explain the software you need to use Linux or MacOS for `kind`
development.
Before you being you must have:

* [GitHub account][github]
* `git`
* `go`
* `Container Engine`

## Task 1. Read the Kubernetes community guidelines
Make sure to read you read the [Kubernetes community guidelines][community].
In specific, read through the [Kubernetes contributor guidelines][contributor].

## Task 2. Install git
Install `git` on your local machine.
You can check if `git` is already on your system and properly installed with 
the following command:

```
$ git --version
```
This documentation is written using `git` version 2.17.1. 
Your version may be different depending on your OS.

## Task 3. Install or upgrade Go
Install or upgrade [Go using the instructions for your operating system][golang].
You can check if Go is in your system with the following command:

```
$ go version
```
This documentation is written using Go version 1.11+.

## Task 4. Install or upgrade Docker or Podman
If you haven't already, install the 
[Docker software using the instructions for your operating system][docker].
or
[Podman software using the instructions for your operating system][podman].
If you have an existing installation, check your version and make sure you have
the latest Docker or Podman

To check if `docker` is has been installed:
```
$ docker --version
```
This documentation is written using Docker version 18.09.0.

To check if `podman` is has been installed:
```
$ podman --version
```
This documentation is written using Podman version 0.12.1.



[github]: https://github.com/
[community]: https://github.com/kubernetes/community
[contributor]: https://github.com/kubernetes/community/blob/master/contributors/guide/README.md
[golang]: https://golang.org/doc/install
[docker]: https://docs.docker.com/install/#supported-platforms
[podman]: https://github.com/containers/libpod/install.md
