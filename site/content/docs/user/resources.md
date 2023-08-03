---
title: "Resources"
menu:
  main:
    parent: "user"
    identifier: "resources"
    weight: 4
toc: true
description: |-
  This page contains references to additional external resources for learning about KIND and how to use it.
---
## Using KIND in CI

The [kind-ci/examples] project is a work-in-progress project to give working
examples of using KIND in various continuous integration environments / platforms.

## Tutorials and Guides

Here are a useful external guides / tutorials covering things not yet covered in our docs:

<!--please do not insert every single guide on the internet here-->
<!--ideally many of these guides will eventually have upstream equivalents on this site-->
<!--in the meantime, content that is not covered upstream in particular can be very helpful here-->

### How to use KIND with MetallLB

https://mauilion.dev/posts/kind-metallb/

### How to Test a Kubernetes PR with KIND

https://mauilion.dev/posts/kind-k8s-testing/

### Using Contour Ingress with KIND

https://projectcontour.io/kindly-running-contour/

### Local Ingress Domains for your Kind Cluster

https://mjpitz.com/blog/2020/10/21/local-ingress-domains-kind/

### Connect directly to Docker-for-Mac containers via IP address

https://golangexample.com/connect-directly-to-docker-for-mac-containers-via-ip-address/

### Developing for Kubernetes with KinD

https://docs.gitlab.com/charts/development/kind/

## KubeCon Talks

The authors have given the following talks relating to KIND:

### Deep Dive: KIND - Benjamin Elder & Antonio Ojea

At KubeCon US 2019 we spoke about [KIND internals and the challenges ahead on the road to 1.0][kind-deep-dive].

{{< youtube id="tT-GiZAr6eQ" class="video-wrapper" >}}

### A Kind Workflow for Contributing to Kubernetes - Benjamin Elder & Duffie Cooley & James Munnelly & Patrick Lang

At KubeCon US 2019 we provided a hands on tutorial [for contributing and testing your Kubernetes code with KIND][kind-workflow-for-contributing-to-kubernetes].

{{< youtube id="BPVO2mcfjJk" class="video-wrapper" >}}

### Testing your K8s apps with KIND - Benjamin Elder & James Munnelly

At KubeCon EU 2019 we spoke about [KIND and testing your Kubernetes Applications][testing-k8s-apps-with-kind].

{{< youtube id="8KtmevMFfxA" class="video-wrapper" >}}

### Deep Dive: Testing SIG - Benjamin Elder & James Munnelly

At KubeCon EU 2019 we spoke about KIND and how we use it to test Kubernetes for the [SIG Testing Deep Dive][sig-testing-deep-dive-kind].

{{< youtube id="6m9frvTxK0o" class="video-wrapper" >}}

### Behind Your PR: How Kubernetes Uses Kubernetes to Run Kubernetes CI - Sen Lu & Benjamin Elder

At KubeCon NA 2018 we spoke with [Sen Lu][@krzyzacy] about The Kubernetes Project's
testing tools and infrastructure, including a brief discussion of KIND and running
it on Kubernetes's Kubernetes-based CI infrastructure.

{{< youtube id="pz0lpl6h-Gc" class="video-wrapper" >}}


[@krzyzacy]: https://github.com/krzyzacy
[kind-ci/examples]: https://github.com/kind-ci/examples
[testing-k8s-apps-with-kind]: https://kccnceu19.sched.com/event/MPYy/testing-your-k8s-apps-with-kind-benjamin-elder-google-james-munnelly-jetstackio
[sig-testing-deep-dive-kind]: https://kccnceu19.sched.com/event/MPkC/deep-dive-testing-sig-benjamin-elder-google-james-munnelly-jetstack
[kind-deep-dive]: https://kccncna19.sched.com/event/Uah7/deep-dive-kind-benjamin-elder-google-antonio-ojea-garcia-suse
[kind-workflow-for-contributing-to-kubernetes]: https://kccncna19.sched.com/event/Uaek/tutorial-a-kind-workflow-for-contributing-to-kubernetes-benjamin-elder-google-duffie-cooley-vmware-james-munnelly-jetstack-patrick-lang-microsoft-limited-available-seating-first-come-first-served-basis
