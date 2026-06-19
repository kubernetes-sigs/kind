FROM kindest/node:v1.35.5

# Init feature gates
ENV CLUSTER_TOPOLOGY=true
ENV CLUSTERCTL_DISABLE_VERSIONCHECK=true
ENV EXP_MACHINE_POOL=true
ENV CAPA_EKS_ADD_ROLES=true

# Core tool/version args
ARG CLUSTERCTL=v1.10.10
ENV CLUSTERAWSADM=v2.9.3
ENV HELM=v3.19.0
ENV AWSCLI_VERSION=2.30.6
ARG CAPA=v2.9.3
ARG CAPG=1.6.1-0.4.0
ARG CAPZ=v1.21.3
ENV CAPI_REPO=/root/.cluster-api/local-repository

## System packages (minimal, drop previous Python pip stack)
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl ca-certificates tar gzip unzip less groff \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

## AWS CLI v2 (self-contained)
RUN curl -Ls "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-${AWSCLI_VERSION}.zip" -o /tmp/awscliv2.zip \
    && unzip /tmp/awscliv2.zip -d /tmp/aws-install \
    && /tmp/aws-install/aws/install --bin-dir /usr/local/bin --update \
    && rm -rf /tmp/awscliv2.zip /tmp/aws-install

## Add aliases
RUN echo 'alias k="kubectl"' >> /root/.bash_aliases \
    && echo 'alias capi-logs="kubectl -n capi-system logs -f deploy/capi-controller-manager"' >> /root/.bash_aliases \
    && echo 'alias capa-logs="kubectl -n capa-system logs -f deploy/capa-controller-manager"' >> /root/.bash_aliases \
    && echo 'alias capg-logs="kubectl -n capg-system logs -f deploy/capg-controller-manager"' >> /root/.bash_aliases \
    && echo 'alias capz-logs="kubectl -n capz-system logs -f deploy/capz-controller-manager"' >> /root/.bash_aliases \
    && echo 'alias kc-logs="kubectl -n kube-system logs -f deploy/keoscluster-controller-manager"' >> /root/.bash_aliases \
    && echo 'alias kw="kubectl --kubeconfig /kind/worker-cluster.kubeconfig"' >> /root/.bash_aliases \
    && echo 'if [ -f ~/.bash_aliases ]; then . ~/.bash_aliases; fi' >> /root/.bashrc

# Download clusterctl
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/clusterctl-linux-amd64 -o /usr/local/bin/clusterctl \
    && chmod +x /usr/local/bin/clusterctl

# Download clusterawsadm
RUN curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CLUSTERAWSADM}/clusterawsadm-linux-amd64 -o /usr/local/bin/clusterawsadm \
    && chmod +x /usr/local/bin/clusterawsadm

## Layer cleanup: remove miscellaneous caches to reduce image size & potential vuln surface
RUN rm -rf /root/.cache /tmp/* || true
## TODO (follow-up): Evaluate removing docker CLI / legacy components if not required at runtime
## TODO (follow-up): Consider multi-stage approach to further slim image
# Download helm
## Install helm and plugin (git needed temporarily)
RUN set -e pipefail \
    && apt-get update \
    && apt-get install -y --no-install-recommends git \
    && curl -L https://get.helm.sh/helm-${HELM}-linux-amd64.tar.gz -o /root/helm.tar.gz \
    && tar -xf /root/helm.tar.gz -C /root \
    && mv /root/linux-amd64/helm /usr/local/bin/helm \
    && rm -rf /root/linux-amd64 /root/helm.tar.gz \
    && chmod +x /usr/local/bin/helm \
    && helm plugin install https://github.com/hypnoglow/helm-s3.git \
    && helm plugin list \
    && apt-get purge -y git \
    && apt-get autoremove -y --purge \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create helm charts directory
RUN mkdir -p /stratio/helm 

# Prepare cluster-api private repository
RUN mkdir -p ${CAPI_REPO}/infrastructure-aws/${CAPA} ${CAPI_REPO}/infrastructure-gcp/${CAPG} ${CAPI_REPO}/infrastructure-azure/${CAPZ} ${CAPI_REPO}/cluster-api/${CLUSTERCTL} ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL} ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL} ${CROSSPLANE_CACHE} \
  && echo "providers:" > /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: aws\n    url: ${CAPI_REPO}/infrastructure-aws/${CAPA}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: gcp\n    url: ${CAPI_REPO}/infrastructure-gcp/${CAPG}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: azure\n    url: ${CAPI_REPO}/infrastructure-azure/${CAPZ}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: kubeadm\n    url: ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/bootstrap-components.yaml\n    type: BootstrapProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: kubeadm\n    url: ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/control-plane-components.yaml\n    type: ControlPlaneProvider" >> /root/.cluster-api/clusterctl.yaml \
  && echo "  - name: cluster-api\n    url: ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/core-components.yaml\n    type: CoreProvider" >> /root/.cluster-api/clusterctl.yaml

# Download cluster-api artifacts
RUN for i in metadata.yaml infrastructure-components.yaml; do \
      curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CAPA}/${i} -o ${CAPI_REPO}/infrastructure-aws/${CAPA}/${i} \
      && curl -L https://github.com/Stratio/cluster-api-provider-gcp/releases/download/${CAPG}/${i} -o ${CAPI_REPO}/infrastructure-gcp/${CAPG}/${i} \
      && curl -L https://github.com/kubernetes-sigs/cluster-api-provider-azure/releases/download/${CAPZ}/${i} -o ${CAPI_REPO}/infrastructure-azure/${CAPZ}/${i}; done

RUN curl -L  https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/core-components.yaml -o ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/core-components.yaml \
    && curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/bootstrap-components.yaml -o ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/bootstrap-components.yaml \
    && curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/control-plane-components.yaml -o ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/control-plane-components.yaml \
    && curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/metadata.yaml -o ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml \
    && cp ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/metadata.yaml \
    && cp ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/metadata.yaml
