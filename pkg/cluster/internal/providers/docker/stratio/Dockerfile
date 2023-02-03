FROM kindest/node:v1.24.6

# Tools versions
ENV CLUSTERCTL=v1.3.2
ENV CLUSTERAWSADM=v2.0.2

# Init feature gates
ENV CLUSTER_TOPOLOGY="true"

# Download clusterctl (61.5M)
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# Download clusterawsadm (92.5M)
RUN curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CLUSTERAWSADM}/clusterawsadm-linux-amd64 -o /usr/local/bin/clusterawsadm \
    && chmod +x /usr/local/bin/clusterawsadm

# Add alias for kubectl
RUN echo 'alias k="kubectl"' >> ~/.bashrc
