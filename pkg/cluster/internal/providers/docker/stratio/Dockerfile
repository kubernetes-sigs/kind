FROM kindest/node:v1.27.0

# Init feature gates
ENV CLUSTER_TOPOLOGY=true
ENV CLUSTERCTL_DISABLE_VERSIONCHECK=true

# Tools versions
ENV CLUSTERCTL=v1.4.1
ENV CLUSTERAWSADM=v2.0.2
ENV HELM_VERSION=v3.11.3

# Helm charts
ENV CLUSTER_AUTOSCALER=9.28.0
ENV TIGERA_OPERATOR=v3.25.1

# Cluster-api artifacts
ENV CAPI_REPO=/root/.cluster-api/local-repository
ENV CAPA=v2.0.2

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
  && for i in {1..3}; do timeout 5 helm pull cluster-autoscaler --version ${CLUSTER_AUTOSCALER} --repo https://kubernetes.github.io/autoscaler --untar --untardir /stratio/helm; done \
  && helm pull tigera-operator --version ${TIGERA_OPERATOR} --repo https://docs.projectcalico.org/charts --untar --untardir /stratio/helm

RUN mkdir -p ${CAPI_REPO}/infrastructure-aws/${CAPA} \
  && echo "\
providers:\n\
  - name: "aws"\n\
    url: ${CAPI_REPO}/infrastructure-aws/${CAPA}/infrastructure-components.yaml\n\
    type: "InfrastructureProvider"\n\
" > /root/.cluster-api/clusterctl.yaml \
  && curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CAPA}/metadata.yaml \
    -o ${CAPI_REPO}/infrastructure-aws/${CAPA}/metadata.yaml \
  && curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CAPA}/infrastructure-components.yaml \
    -o ${CAPI_REPO}/infrastructure-aws/${CAPA}/infrastructure-components.yaml
