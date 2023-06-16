FROM kindest/node:v1.27.0

# Init feature gates
ENV CLUSTER_TOPOLOGY=true
ENV CLUSTERCTL_DISABLE_VERSIONCHECK=true

# Tools versions
ENV CLUSTERCTL=v1.4.3
ENV CLUSTERAWSADM=v2.0.2
ENV HELM=v3.11.3

# Helm charts
ENV AZUREDISK_CSI_DRIVER_CHART=v1.27.1
ENV CLOUD_PROVIDER_AZURE_CHART=v1.27.1
ENV CLUSTER_AUTOSCALER_CHART=9.28.0
ENV TIGERA_OPERATOR_CHART=v3.25.1

# Cluster-api artifacts
ENV CAPI_REPO=/root/.cluster-api/local-repository
ENV CAPA=v2.0.2
ENV CAPG=v1.3.0
ENV CAPZ=v1.9.2

# Install vim
RUN apt-get update && apt-get install -y \
  vim \
  && rm -rf /var/lib/apt/lists/*

# Add aliases
RUN echo 'alias k="kubectl"' >> ~/.bash_aliases \
    && echo 'alias capi-logs="kubectl -n capi-system logs -f deploy/capi-controller-manager"' >> ~/.bash_aliases \
    && echo 'alias capa-logs="kubectl -n capa-system logs -f deploy/capa-controller-manager"' >> ~/.bash_aliases \
    && echo 'alias capg-logs="kubectl -n capg-system logs -f deploy/capg-controller-manager"' >> ~/.bash_aliases \
    && echo 'alias capz-logs="kubectl -n capz-system logs -f deploy/capz-controller-manager"' >> ~/.bash_aliases \
    && echo 'alias kw="kubectl --kubeconfig /kind/worker-cluster.kubeconfig"' >> ~/.bash_aliases

# Download clusterctl
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# Download clusterawsadm
RUN curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CLUSTERAWSADM}/clusterawsadm-linux-amd64 -o /usr/local/bin/clusterawsadm \
    && chmod +x /usr/local/bin/clusterawsadm

# Download helm
RUN curl -L https://get.helm.sh/helm-${HELM}-linux-amd64.tar.gz -o /root/helm.tar.gz \
  && tar -xf /root/helm.tar.gz -C /root && mv /root/linux-amd64/helm /usr/local/bin/helm \
  && rm -rf /root/linux-amd64 /root/helm.tar.gz \
  && chmod +x /usr/local/bin/helm

# Download helm charts
RUN mkdir -p /stratio/helm \
  && for i in $(seq 1 3); do timeout 5 helm pull azuredisk-csi-driver --version ${AZUREDISK_CSI_DRIVER_CHART} --repo https://raw.githubusercontent.com/kubernetes-sigs/azuredisk-csi-driver/master/charts --untar --untardir /stratio/helm && break; done \
  && for i in $(seq 1 3); do timeout 5 helm pull cloud-provider-azure --version ${CLOUD_PROVIDER_AZURE_CHART} --repo https://raw.githubusercontent.com/kubernetes-sigs/cloud-provider-azure/master/helm/repo --untar --untardir /stratio/helm && break; done \
  && for i in $(seq 1 3); do timeout 5 helm pull cluster-autoscaler --version ${CLUSTER_AUTOSCALER_CHART} --repo https://kubernetes.github.io/autoscaler --untar --untardir /stratio/helm && break; done \
  && for i in $(seq 1 3); do timeout 5 helm pull tigera-operator --version ${TIGERA_OPERATOR_CHART} --repo https://docs.projectcalico.org/charts --untar --untardir /stratio/helm && break; done

# Prepare cluster-api offline repository
RUN mkdir -p ${CAPI_REPO}/infrastructure-aws/${CAPA} ${CAPI_REPO}/infrastructure-gcp/${CAPG} ${CAPI_REPO}/infrastructure-azure/${CAPZ} \
  && echo "providers:" > /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: aws\n    url: ${CAPI_REPO}/infrastructure-aws/${CAPA}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: gcp\n    url: ${CAPI_REPO}/infrastructure-gcp/${CAPG}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: azure\n    url: ${CAPI_REPO}/infrastructure-azure/${CAPZ}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml

# Download cluster-api artifacts
RUN for i in metadata.yaml infrastructure-components.yaml; do \
      curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CAPA}/${i} -o ${CAPI_REPO}/infrastructure-aws/${CAPA}/${i} \
      && curl -L https://github.com/kubernetes-sigs/cluster-api-provider-gcp/releases/download/${CAPG}/${i} -o ${CAPI_REPO}/infrastructure-gcp/${CAPG}/${i} \
      && curl -L https://github.com/kubernetes-sigs/cluster-api-provider-azure/releases/download/${CAPZ}/${i} -o ${CAPI_REPO}/infrastructure-azure/${CAPZ}/${i}; done
