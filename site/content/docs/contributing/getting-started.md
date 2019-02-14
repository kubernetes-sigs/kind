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

### Install Documentation Tools

If you wish to contribute to the documentation, it is recommended but not 
required to install [hugo], which we use to develop this site.

Please see: https://gohugo.io/getting-started/installing/

### Install Developer Tools

If you wish to contribute to kind's code you will need to install the following:

* `git`
* `go`
* `docker`

#### Install git
Install `git` on your local machine.
You can check if `git` is already on your system and properly installed with 
the following command:

```
git --version
```
This documentation is written using `git` version 2.17.1. 
Your version may be different depending on your OS.

#### Install or upgrade Go
Install or upgrade [Go using the instructions for your operating system][golang].
You can check if Go is in your system with the following command:

```
go version
```
This documentation is written using Go version 1.11+.

#### Install or upgrade Docker
If you haven't already, install the 
[Docker software using the instructions for your operating system][docker].
If you have an existing installation, check your version and make sure you have
the latest Docker.

To check if `docker` is has been installed:
```
docker --version
```
This documentation is written using Docker version 18.09.2.

## 3. Read The Docs 

The [design principles], [1.0 roadmap], [project structure], and [initial design]
may be helpful to review before contributing.

## 4. Reaching Out

Issues are tracked on GitHub. Please check [the issue tracker][issues] to see
if there is any existing dicussion or work related to your interests.

If you do not see anything, please [file a new issue][file an issue].

Please reach out for bugs, feature requests, and other issues!  
The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel
- [filing an issue][file an issue]
- The Kubernetes [SIG-Testing Mailing List]

Current maintainers are [@BenTheElder] and [@munnerz] - feel free to
reach out if you have any questions!

See also: the Kubernetes [community page].

[hugo]: https://gohugo.io
[issues]: https://github.com/kubernetes-sigs/kind/issues
[file an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[design principles]: /docs/design/principles
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[project scope]: /docs/contributing/project-scope
[project structure]: /docs/devel/project-structure
[initial design]: /docs/design/initial
[github]: https://github.com/
[golang]: https://golang.org/doc/install
[docker]: https://docs.docker.com/install/#supported-platforms
[community]: https://github.com/kubernetes/community
[contributor]: https://github.com/kubernetes/community/blob/master/contributors/guide/README.md
[Kubernetes Slack]: http://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[@BenTheElder]: https://github.com/BenTheElder
[@munnerz]: https://github.com/munnerz
[community page]: http://kubernetes.io/community/
