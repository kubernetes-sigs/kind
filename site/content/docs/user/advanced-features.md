---
title: "Advanced Features"
menu:
  main:
    parent: "user"
    identifier: "advanced-features"
    weight: 3
---
# Advanced Features


## Building The Base Image
To build the `base-image` we use the `build` command:
```
kind build base-image
```

If you want to specify the path to the base image source files you can use the
`--source` flag.

If `--source` is not specified, kind will attempt to automatically locate
the `images/base` base source directory.

By default, the base image will be tagged as `kindest/base:latest`.
If you want to change this, you can use the `--image` flag.

```
kind build base-image --image base:v0.1.0
```


#### Mapping ports to the host machine
You can map extra ports from the nodes to the host machine with `extraPortMappings`:
```yaml
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
- role: control-plane
- role: worker
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    listenAddress: "127.0.0.1" # Optional, defaults to "0.0.0.0"
    protocol: udp # Optional, defaults to tcp
```
This can be useful if using `NodePort` services or daemonsets exposing host ports.

### Enable Feature Gates in Your Cluster

Feature gates are a set of key=value pairs that describe alpha or experimental features. In order to enable a gate you have to [customize your kubeadm configuration][customize control plane with kubeadm], and it will depend on what gate and component you want to enable. An example kind config can be:

```
# this config file contains all config fields with comments
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
# patch the generated kubeadm config with some extra settings
kubeadmConfigPatches:
- |
  apiVersion: kubeadm.k8s.io/v1beta2
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      "feature-gates": "FeatureGateName=true"
  scheduler:
    extraArgs:
      "feature-gates": "FeatureGateName=true"
  controllerManager:
    extraArgs:
      "feature-gates": "FeatureGateName=true"
- |
  apiVersion: kubeadm.k8s.io/v1beta2
  kind: InitConfiguration
  metadata:
    name: config
  nodeRegistration:
    kubeletExtraArgs:
      "feature-gates": "FeatureGateName=true"
# 1 control plane node and 3 workers
nodes:
# the control plane node config
- role: control-plane
# the three workers
- role: worker
```

#### IPv6 clusters
You can run ipv6 only clusters using `kind`, but first you need to
[enable ipv6 in your docker daemon][docker enable ipv6].

```yaml
# an ipv6 cluster
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  ipFamily: ipv6
nodes:
# the control plane node
- role: control-plane
- role: worker
- role: worker
```

### Configure kind to use a proxy
If you are running kind in an environment that requires a proxy, you may need to configure kind to use it.

You can configure kind to use a proxy using one or more of the following [environment variables][proxy environment variables] (uppercase takes precedence):

* HTTP_PROXY or http_proxy
* HTTPS_PROXY or https_proxy
* NO_PROXY or no_proxy

**Note**: If you set a proxy it would be used for all the connection requests.
It's important that you define what addresses doesn't need to be proxied with the NO_PROXY variable, typically you should avoid to proxy your docker network range `NO_PROXY=172.17.0.0/16`

### Exporting Cluster Logs
kind has the ability to export all kind related logs for you to explore.
To export all logs from the default cluster (context name `kind`):
```
kind export logs
Exported logs to: /tmp/396758314
```

Like all other commands, if you want to perform the action on a cluster with a
different context name use the `--name` flag.

As you can see, kind placed all the logs for the cluster `kind` in a
temporary directory. If you want to specify a location then simply add the path
to the directory after the command:
```
kind export logs ./somedir  
Exported logs to: ./somedir
```

The structure of the logs will look more or less like this:
```
.
├── docker-info.txt
└── kind-control-plane/
    ├── containers
    ├── docker.log
    ├── inspect.json
    ├── journal.log
    ├── kubelet.log
    ├── kubernetes-version.txt
    └── pods/
```
The logs contain information about the Docker host, the containers running 
kind, the Kubernetes cluster itself, etc.

[proxy environment variables]: https://docs.docker.com/network/proxy/#use-environment-variables
[customize control plane with kubeadm]: https://kubernetes.io/docs/setup/independent/control-plane-flags/
[docker enable ipv6]: https://docs.docker.com/config/daemon/ipv6/
