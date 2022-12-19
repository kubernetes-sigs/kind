FROM kindest/node:v1.24.6

# Download clusterctl (61.5M)

RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.2.4/clusterctl-linux-amd64 -o clusterctl
RUN chmod +x ./clusterctl
RUN mv ./clusterctl /usr/local/bin/clusterctl

# Download clusterawsadm (92.5M)

RUN curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/v1.5.0/clusterawsadm-linux-amd64 -o clusterawsadm
RUN chmod +x clusterawsadm
RUN mv clusterawsadm /usr/local/bin

# Init feature gates

ENV CLUSTER_TOPOLOGY="true"
