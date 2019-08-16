---
title: "Getting Started"
menu:
  main:
    parent: "contributing"
    identifier: "getting started"
    weight: 1
---
# Getting Started

Welcome! This guide covers how to get started contributing to kind.

## 1. Read the Kubernetes community guidelines

Make sure to read you read the [Kubernetes community guidelines][community].
In specific, read through the [Kubernetes contributor guidelines][contributor].

Additionally, note that kind is developed on [GitHub][github] and will require
an account to contribute.

## 2. Install Tools

### Install git

Our source code is managed with [`git`][git], to develop locally you
will need to install `git`.

You can check if `git` is already on your system and properly installed with 
the following command:

```
git --version
```

### Install Hugo

If you wish to contribute to the documentation, it is recommended but not 
required to install [hugo], which we use to develop this site.

Please see: https://gohugo.io/getting-started/installing/


### Install Go

To work on kind's codebase you will need [Go][golang].

Install or upgrade [Go using the instructions for your operating system][golang].
You can check if Go is in your system with the following command:

```
go version
```

Preferably Go `1.12.9` or greater should be installed. 

Correct automatic formatting of the source with `gofmt` requires at least
`1.11.0`.

Working with Go [modules] (which we use for dependency management) requires at
least `1.11.4` due to checksum bugs in lower versions.

### Install Docker

Currently, to create clusters you will need to install [Docker][docker].

If you haven't already, [install Docker][install docker], following the
[official instructions][install docker].
If you have an existing installation, check your version and make sure you have
the latest Docker.

To check if `docker` has has been installed:
```
docker --version
```
This documentation is written using Docker version 18.09.2.

## 3. Read The Docs 

The [design principles], [1.0 roadmap], [project structure], and [initial design]
may be helpful to review before contributing.

## 4. Reaching Out

Issues are tracked on GitHub. Please check [the issue tracker][issues] to see
if there is any existing discussion or work related to your interests.

If you do not see anything, please [file a new issue][file an issue].

Please reach out for bugs, feature requests, and other issues!  
The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel
- [filing an issue][file an issue]
- The Kubernetes [SIG-Testing Mailing List]

Current maintainers are [@BenTheElder] and [@munnerz] - feel free to
reach out if you have any questions!

See also: the Kubernetes [community page].

[git]: https://git-scm.com/
[hugo]: https://gohugo.io
[issues]: https://github.com/kubernetes-sigs/kind/issues
[file an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[design principles]: /docs/design/principles
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[project scope]: /docs/contributing/project-scope
[project structure]: /docs/contributing/project-structure
[initial design]: /docs/design/initial
[github]: https://github.com/
[golang]: https://golang.org/doc/install
[docker]: https://www.docker.com/
[install docker]: https://docs.docker.com/install/#supported-platforms
[community]: https://github.com/kubernetes/community
[contributor]: https://github.com/kubernetes/community/blob/master/contributors/guide/README.md
[Kubernetes Slack]: http://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[@BenTheElder]: https://github.com/BenTheElder
[@munnerz]: https://github.com/munnerz
[community page]: http://kubernetes.io/community/
[modules]: https://github.com/golang/go/wiki/Modules
[SIG-Testing Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-testing
