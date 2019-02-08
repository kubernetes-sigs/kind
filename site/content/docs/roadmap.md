# RoadMap 🗺️

New year, new roadmap 🎉  
This document outlines some goals, non-goals, and future aspirations for kind
as a project.

High level goals for kind include:

- Reach [1.0] status / 
stability by the end of Q1 2019 / the Kubernetes 1.14 release cycle
- Support multi-node clusters - [#117] - DONE ✔
- Support offline / air-gapped clusters
  - pre-loaded / offline CNI - [#200]
- Improve Windows support
  - add Windows binaries to releases - [#155] - DONE ✔
  - improve instructions for KUBECONFIG in particular
- Improve documentation
  - expand examples of using kind
  - cover known issues, debugging, work-arounds, etc.
  - create a GitHub-pages landing page with docs
- Improving logging and debuggability
  - enable audit-logging
- Support non-amd64 architectures (namely ARM) - [#166]
- Automated publishing of Kubernetes release based kind "node" images - [#197]

**Non**-Goals include:

- Supporting every possible Kubernetes configuration
  - In order to best support offline / hermetic clusters, we will likely not
  offer many options for CNI etc. out of the box. We may revisit this later.
- Being "production workload ready" - kind is meant to be used:
  - for testing Kubernetes itself
  - for testing against Kubernetes (EG in CI on Travis, Circle, etc.)
  - for "local" clusters on developer machines
  - NOT to host workloads serving user traffic etc.
- Replacing [Phippy] 🦒 -- kind isn't trying to replace all the things
and Phippy is awesome ❤️

Longer Term goals include:

- Enabling a suitable local storage provider for testing applications that need
persistent storage
- Improving "kind as a library"

Misc:

- setup a regular Zoom meeting for the project - DONE ✔
- achieve certified Kubernetes conformance - DONE ✔

Other goals / tasks not listed here can be found both in [the 1.0 project] and 
more generally triaged for rough-priority in the [GitHub issues].

[1.0]: https://github.com/kubernetes-sigs/kind/projects/1
[the 1.0 project]: https://github.com/kubernetes-sigs/kind/projects/1
[GitHub issues]: https://github.com/kubernetes-sigs/kind/issues
[#117]: https://github.com/kubernetes-sigs/kind/issues/117
[#166]: https://github.com/kubernetes-sigs/kind/issues/166
[#155]: https://github.com/kubernetes-sigs/kind/issues/155
[#197]: https://github.com/kubernetes-sigs/kind/issues/197
[#200]: https://github.com/kubernetes-sigs/kind/issues/200

[Phippy]: https://phippy.io/
