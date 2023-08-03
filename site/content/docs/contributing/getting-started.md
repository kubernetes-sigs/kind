---
title: "Getting Started"
menu:
  main:
    parent: "contributing"
    identifier: "getting started"
    weight: 1
toc: true
description: |-
  Welcome! 👋 

  This guide covers how to start contributing to kind 😄
---

## 1. Familiarize Yourself With Contributing to Kubernetes Projects
### Read the Kubernetes Community Guidelines

Make sure to read you read the [Kubernetes community guidelines][community].
In specific, read through the [Kubernetes contributor guidelines][contributor].

Additionally, note that 

### Setup GitHub Account

Kubernetes and kind are developed on [GitHub][github] and will require
an account to contribute.

### Sign CNCF CLA

The Kubernetes project requires the [CNCF][CNCF] [CLA][CNCF-cla] be signed against
your GitHub account for all contributions in all subprojects.

You'll need to get the CLA signed to contribute.

### Check The Kubernetes Contributor Guides

You may come back to this later, but we highly recommend reading these:

- [Kubernetes Contributor Guide](https://git.k8s.io/community/contributors/guide) 
  - Main contributor documentation, or you can just jump directly to the [contributing section](https://git.k8s.io/community/contributors/guide#contributing)
- [Contributor Cheat Sheet](https://git.k8s.io/community/contributors/guide/contributor-cheatsheet)
   - Common resources for existing developers

## 2. Install Tools

### Install Git

Our source code is managed with [`git`][git], to develop locally you
will need to install `git`.

You can check if `git` is already on your system and properly installed with 
the following command:

```
git --version
```

### Install Docker

Currently, to create clusters you will need to install [Docker][docker].

If you haven't already, [install Docker][install docker], following the
[official instructions][install docker].
If you have an existing installation, check your version and make sure you have
the latest Docker.

To check if `docker` has been installed:
```
docker --version
```
This documentation is written using Docker version 18.09.2.

### Install Go (optional)

KIND is written in [Go][golang], however our makefiles automatically ensure the
correct version of go when building or testing.

You may still wish to install go on your machine to make it easier to integrate
into your editor etc. You can find the version of go we're currently using to develop kind in the [`.go-version`][go-version] file in the kind repo.

Install or upgrade [Go using the instructions for your operating system][golang].
You can check if Go is in your system with the following command:

## 3. Read The Docs 

The [design principles], [1.0 roadmap], and [initial design]
may be helpful to review before contributing. These docs cover some of the project
philosophy and direction.

## 4. Reaching Out

Issues are tracked on GitHub. Please check [the issue tracker][issues] to see
if there is any existing discussion or work related to your interests.

In particular, if you're just getting started, you may want to look for issues
labeled <a href="https://github.com/kubernetes-sigs/kind/labels/good%20first%20issue" class="gh-label" style="background: #7057ff; color: white">good first issue</a> or <a href="https://github.com/kubernetes-sigs/kind/labels/help%20wanted" class="gh-label" style="background: #006b75; color: white">help wanted</a> which are standard labels in the Kubernetes
project.
The <a href="https://github.com/kubernetes-sigs/kind/labels/help%20wanted" class="gh-label" style="background: #006b75; color: white">help wanted</a> label marks issues we're actively seeking help with while <a href="https://github.com/kubernetes-sigs/kind/labels/good%20first%20issue" class="gh-label" style="background: #7057ff; color: white">good first issue</a> is additionally applied to a subset of issues we think will be particularly good for newcomers.

If you're interested in working on any of these, leave a comment to let us know!

If you do not see anything, please [file a new issue][file an issue].

> **NOTE**: _Please_ file an enhancement / [feature request issue][fr-issue] to discuss features before filing a PR (ideally even before writing any code), we have a lot to consider with respect to our
> existing users and future support when accepting any new feature.
>
> To streamline the process, please reach out and discuss the concept and design
> / approach ASAP so the maintainers and community can get involved early.

Also -- Please reach out in general for bugs, feature requests, and other issues!  

The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel (most active, along with the community)
- The issue tracker by [filing an issue][file an issue]
- The Kubernetes [SIG-Testing][SIG-Testing] [Mailing List][SIG-Testing Mailing List]

Current maintainers are [@aojea] and [@BenTheElder] -- feel free to
reach out directly if you have any questions!

See also: the Kubernetes [community page].

## 5. Next Steps

Okay, so you've gotten your development environment setup, you've read all the
contributor guides, signed the CLA ... now what?

If you're planning to contribute code changes, you'll want to read the [development guide] next.

If you're looking to contribute documentation improvements, first: Thank you! 🎉🤗
You'll specifically want to see the [documentation section] of the development guide.

[git]: https://git-scm.com/
[hugo]: https://gohugo.io
[issues]: https://github.com/kubernetes-sigs/kind/issues
[file an issue]: https://github.com/kubernetes-sigs/kind/issues/new/choose
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
[Kubernetes Slack]: https://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[@BenTheElder]: https://github.com/BenTheElder
[@aojea]: https://github.com/aojea
[community page]: https://kubernetes.io/community/
[modules]: https://github.com/golang/go/wiki/Modules
[SIG-Testing Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-testing
[CNCF]: https://www.cncf.io/
[CNCF-cla]: https://git.k8s.io/community/CLA.md
[fr-issue]: https://github.com/kubernetes-sigs/kind/issues/new?labels=kind%2Ffeature&template=enhancement.md
[SIG-Testing]: https://github.com/kubernetes/community/blob/master/sig-testing/README.md
[go-version]: https://sigs.k8s.io/kind/.go-version
[development guide]: /docs/contributing/development
[documentation section]: /docs/contributing/development#documentation
