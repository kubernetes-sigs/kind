---
title: "Configuration"
menu:
  main:
    parent: "user"
    identifier: "user-configuration"
    weight: 3
---

<h1>⚠️ This page is a stub! We're working on it! ⚠️ </h1>

# Configuration

This guide covers how to configure KIND cluster creation.

We know this is currently a bit lacking right now and will expand it over time - PRs welcome!

<!--TODO: TOC?-->

## Getting Started

To configure kind cluster creation, you will need to create a [YAML] config file.
This file follows Kubernetes conventions for versioning etc. <!--todo links for this-->

A minimal valid config is:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
```

This config merely specifies that we are configuration a KIND cluster (`kind: Cluster`)
and that the version of KIND's config we are using is `v1alpha4` (`apiVersion: kind.x-k8s.io/v1alpha4`).

Any given version of kind may support different versions which will have different
options and behavior. This is why we must always specify the version.

This mechanism is inspired by Kubernetes resources and component config.

To use this config, place the contents in a file `config.yaml` and then run
`kind create cluster --config=config.yaml` from the same directory.

You can also include a full file path like `kind create cluster --config=/foo/bar/config.yaml`.

## Cluster-Wide Options

The following high level options are available.

NOTE: not all options are documented yet!  We will fix this with time, PRs welcome!

### Networking

Multiple details of the cluster's networking can be customized under the
`networking` field.

#### IP Family

KIND has limited support for IPv6 (and soon dual-stack!) clusters, you can switch
from the default of IPv4 by setting:


{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  ipFamily: ipv6
{{< /codeFromInline >}}

NOTE: you may need to reconfigure your docker daemon to enable ipv6 in order
to use this. 

IPv6 does not work on docker for mac because port forwarding ipv6
is not yet supported in docker for mac.

#### API Server

The API Server listen address and port can be customized with:
{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  # WARNING: It is _strongly_ recommended that you keep this the default
  # (127.0.0.1) for security reasons. However it is possible to change this.
  apiServerAddress: "127.0.0.1"
  # By default the API server listens on a random open port.
  # You may choose a specific port but probably don't need to in most cases.
  # Using a random port makes it easier to spin up multiple clusters.
  apiServerPort: 6443
{{< /codeFromInline  >}}

{{< securitygoose >}}**NOTE**: You should really think thrice before exposing your kind cluster publicly!
kind does not ship with state of the art security or any update strategy (other than
disposing your cluster and creating a new one)! We strongly discourage exposing kind
to anything other than loopback.{{</ securitygoose >}}

#### Pod Subnet

You can configure the subnet used for pod IPs by setting

{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  podSubnet: "10.244.0.0/16"
{{< /codeFromInline >}}

#### Service Subnet

You can configure the Kubernetes service subnet used for service IPs by setting

{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  serviceSubnet: "10.96.0.0/12"
{{< /codeFromInline >}}

#### Disable Default CNI

KIND ships with a simple networking implementation ("kindnetd") based around
standard CNI plugins (`ptp`, `host-local`, ...) and simple netlink routes.

This CNI also handles IP masquerade.

You may disable the default to install a different CNI. This is a power user
feature with limited support, but many common CNI manifests are known to work,
e.g. Calico.
{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  # the default CNI will not be installed
  disableDefaultCNI: true
{{< /codeFromInline >}}


### Nodes
The `kind: Cluster` object has a `nodes` field containing a list of `node`
objects. If unset this defaults to:

```yaml
nodes:
# one node hosting a control plane
- role: control-plane
```

You can create a multi node cluster with the following config:

{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
# One control plane node and three "workers".
#
# While these will not add more real compute capacity and
# have limited isolation, this can be useful for testing
# rolling updates etc.
#
# The API-server and other control plane components will be
# on the control-plane node.
#
# You probably don't need this unless you are testing Kubernetes itself.
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
{{< /codeFromInline >}}

You can also set a specific Kubernetes version by setting the `node`'s container image. You can find available image tags on the [releases page](https://github.com/kubernetes-sigs/kind/releases). Please include the `@sha256:` [image digest](https://docs.docker.com/engine/reference/commandline/pull/#pull-an-image-by-digest-immutable-identifier) from the image in the release notes, as seen in this example:

{{< codeFromInline lang="yaml" >}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
  image: kindest/node:v1.16.4@sha256:b91a2c2317a000f3a783489dfb755064177dbc3a0b2f4147d50f04825d016f55
{{< /codeFromInline >}}

### Authentication

#### OpenID Connect

You can set up OpenID Connect (OIDC) authentication by the following steps:

1. Deploy an OIDC provider outside of the cluster.
1. Create a cluster with the extra arguments for OIDC.
1. Set up kubectl, e.g. [kubelogin](https://github.com/int128/kubelogin).

You need to consider the following constraints:

- The issuer URL must be HTTPS. A TLS certificate is required.
- You cannot deploy an OIDC provider in the cluster,
  because the API server cannot access any pod or service.
- The API server and kubectl access an OIDC provider via the same URL.

Here is an example of cluster config.

{{< codeFromFile file="static/examples/config-with-oidc.yaml" lang="yaml" >}}

You can use [Dex](https://github.com/dexidp/dex) as an OIDC provider.
Here is a minimum config of Dex.

{{< codeFromFile file="static/examples/oidc/dex.yaml" lang="yaml" >}}

You can set up the authentication using Dex on Docker as follows:

```sh
# run a container of Dex
docker run -d --name dex-server -p 10443:10443 -v $PWD:/cfg quay.io/dexidp/dex:v2.21.0 serve /cfg/dex.yaml

# create a cluster
kind create cluster --config config-with-oidc.yaml

# set up the hosts so that the API server can access Dex
docker inspect -f '{{.NetworkSettings.IPAddress}}' dex-server | sed -e 's,$, dex-server,' | \
  kubectl -n kube-system exec -i kube-apiserver-kind-control-plane -- tee -a /etc/hosts

# set up the hosts so that kubectl can access Dex
echo '127.0.0.1 dex-server' | sudo tee -a /etc/hosts

# bind a cluster role to your user
kubectl create clusterrolebinding oidc-admin --clusterrole=cluster-admin --user=admin@example.com

# set up the kubeconfig (using kubelogin)
kubectl config set-credentials oidc --exec-api-version=client.authentication.k8s.io/v1beta1 \
  --exec-command=kubectl \
  --exec-arg=oidc-login \
  --exec-arg=get-token \
  --exec-arg=--oidc-issuer-url=https://dex-server:10443/dex \
  --exec-arg=--oidc-client-id=YOUR_CLIENT_ID \
  --exec-arg=--oidc-client-secret=YOUR_CLIENT_SECRET \
  --exec-arg=--oidc-extra-scope=email \
  --exec-arg=--certificate-authority=$PWD/dex-ca.crt

# make sure you can access the cluster
kubectl --user=oidc cluster-info
```


## Per-Node Options

The following options are available for setting on each entry in `nodes`.

NOTE: not all options are documented yet!  We will fix this with time, PRs welcome!

### Extra Mounts

Extra mounts can be used to pass through storage on the host to a kind node
for persisting data, mounting through code etc.

{{< codeFromFile file="static/examples/config-with-mounts.yaml" lang="yaml" >}}


### Extra Port Mappings

Extra port mappings can be used to port forward to the kind nodes. This is a 
cross-platform option to get traffic into your kind cluster. 

With docker on Linux you can simply send traffic to the node IPs from the host
without this, but to cover macOS and Windows you'll want to use these.

You may also want to see the [Ingress Guide].

{{< codeFromFile file="static/examples/config-with-port-mapping.yaml" lang="yaml" >}}


[Ingress Guide]: ./../ingress
