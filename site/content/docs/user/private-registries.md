---
title: "Private Registries"
menu:
  main:
    parent: "user"
    identifier: "user-private-registries"
    weight: 3
---
# Private Registries

Some users may want to test applications on kind that require pulling images
from authenticated private registries, there are multiple ways to do this.


## Use ImagePullSecrets

Kubernetes supports configuring pods to use `imagePullSecrets` for pulling
images. If possible, this is the preferable and most portable route.

See [the upstream kubernetes docs for this][imagePullSecrets],
kind does not require any special handling to use this.

If you already have the config file locally but would still like to use secrets,
read through kubernetes' docs for [creating a secret from a file][imagePullFileSecrets].

## Pull to the Host and Side-Load

kind can [load an image][loading an image] from the host with the `kind load ...`
commands. If you configure your host with credentials to pull the desired 
image(s) and then load them to the nodes you can avoid needing to authenticate 
on the nodes.


# Add Credentials to the Nodes

Generally the upstream docs for [using a private registry] apply, with kind
there are two options for this.

## Mount a Config File to Each Node

If you pre-create a docker config.json containing credential(s) on the host
you can mount it to each kind node.

Assuming your file is at `/path/to/my/secret.json`, the kind config would be:

```yaml
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
- role: control-plane
  extraMounts:
  - containerPath: /var/lib/kubelet/config.json
    hostPath: /path/to/my/secret.json
```

### Use an Access Token

A credential can be programmatically added to the nodes at runtime.

If you do this then kubelet must be restarted on each node to pick up the new credentials.

An example shell snippet for generating a [gcr.io][GCR] cred file on your host machine
using Access Tokens:

{{< codeFromFile file="static/examples/kind-gcr.sh" >}}

### Use a Service Account

Access tokens are short lived, so you may prefer to use a Service Account and keyfile instead.
First, either download the key from the console or generate one with gcloud:

```
gcloud iam service-accounts keys create <output.json> --iam-account <account email>
```

Then, replace the `gcloud auth print-access-token | ...` line from the [access token snippet](#use-an-access-token) with:

```
cat <output.json> | docker login -u _json_key --password-stdin https://gcr.io
```

See Google's [upstream docs][keyFileAuthentication] on key file authentication for more details.

[keyFileAuthentication]: https://cloud.google.com/container-registry/docs/advanced-authentication#json_key_file
[imagePullSecrets]: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod
[imagePullFileSecrets]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials
[loading an image]: /docs/user/quick-start/#loading-an-image-into-your-cluster
[using a private registry]: https://kubernetes.io/docs/concepts/containers/images/#using-a-private-registry
[GCR]: https://cloud.google.com/container-registry/
