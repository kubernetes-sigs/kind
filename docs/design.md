# Design

This is the root design documentation for `kind`. See also the project
[README.md](./../README.md).

## Overview

`kind` or **k**ubernetes **in** **d**docker is a suite of tooling for local kubernetes
"clusters" where each "node" in the cluster is a docker container running on the
same local machine.

kind is divided into go packages implementing most of the functionality, a
command line for users, and a "node" base image. The intent is that the `kind`
the suite of packages should eventually be importable and reusable by other
tools (e.g. [kubetest](https://github.com/kubernetes/test-infra/tree/master/kubetest))
while the CLI provides a quick way to use and debug these packages.

## The Node Image

The ["node" image](./../images/node) is a small-ish Docker image for running
nested containers, systemd, and kubernetes components.

To do this we need to set up an environment that will meet the CRI 
(currently just docker) and systemd's particular needs. Documentation for this
is fairly detailed inline to the image's [Dockerfile](./../images/node/Dockerfile)),
but essentially:

- we preinstall tools / packages expected by systemd/docker/kubernetes other
than Kubernetes itself.

- we install a custom entrypoint that allows us to perform some actions before
the container truly boots

- we set up a systemd service to forward journal logs to the container tty

- we do a few tricks to minimize unnecessary services and inform systemd that it
is in docker. (see the [Dockerfile](./../images/node/Dockerfile))

# TODO(bentheelder): fill in further docs as things stabilize / functionality is checked in, particularly around node management
