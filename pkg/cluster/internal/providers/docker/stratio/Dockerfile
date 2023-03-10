FROM kindest/node:v1.24.7

# Init feature gates
ENV CLUSTER_TOPOLOGY=true

# Tools versions
ENV CLUSTERCTL=v1.3.2
ENV CLUSTERAWSADM=v2.0.2
ENV HELM_VERSION=v3.11.0

# Helm charts
ENV CLUSTER_AUTOSCALER=9.25.0
ENV TIGERA_OPERATOR=v3.22.3

# Download clusterctl (61.5M)
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# Download clusterawsadm (92.5M)
RUN curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CLUSTERAWSADM}/clusterawsadm-linux-amd64 -o /usr/local/bin/clusterawsadm \
    && chmod +x /usr/local/bin/clusterawsadm

# Add alias for kubectl
RUN echo 'alias k="kubectl"' >> ~/.bashrc

RUN curl -L https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz -o /root/helm.tar.gz \
  && tar -xf /root/helm.tar.gz -C /root && mv /root/linux-amd64/helm /usr/local/bin/helm \
  && rm -rf /root/linux-amd64 \
  && rm /root/helm.tar.gz \
  && chmod +x /usr/local/bin/helm

RUN mkdir -p /stratio/helm \
  && until timeout 5 helm pull cluster-autoscaler --version ${CLUSTER_AUTOSCALER} --repo https://kubernetes.github.io/autoscaler --untar --untardir /stratio/helm; do sleep 2; done \
  && helm pull tigera-operator --version ${TIGERA_OPERATOR} --repo https://docs.projectcalico.org/charts --untar --untardir /stratio/helm
