FROM debian:bookworm-slim

ARG UPGRADE_DIR=upgrade/resources/scripts

# =========================
# Environment configuration
# =========================
ENV CLUSTER_TOPOLOGY=true \
    CLUSTERCTL_DISABLE_VERSIONCHECK=true \
    CAPI_REPO=/root/.cluster-api/local-repository \
    AWS_CONFIG_FILE=/upgrade/.aws/config \
    AWS_SHARED_CREDENTIALS_FILE=/upgrade/.aws/credentials \
    KUBECONFIG=/upgrade/.kube/config \
    # Disable pagers for all tools \
    PAGER=cat \
    AWS_PAGER="" \
    GIT_PAGER=cat \
    # Azure CLI non-interactive \
    AZURE_CORE_NO_COLOR=true \
    AZURE_CORE_OUTPUT=json \
    # Debian non-interactive \
    DEBIAN_FRONTEND=noninteractive

# Tool versions
ARG CLUSTERCTL=v1.10.10
ARG YQ_VERSION=v4.45.1
ARG KUBECTL_VERSION=1.32.12
ARG HELM_VERSION=3.15.2
ARG CAPA=v2.9.3
ARG CAPG=1.6.1-0.4.0
ARG CAPZ=v1.21.3

# =========================
# Base system dependencies
# =========================
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    curl \
    jq \
    git \
    vim \
    ca-certificates \
    unzip \
    apt-transport-https \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

# =========================
# Install yq
# =========================
RUN curl -L https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 \
    -o /usr/local/bin/yq && \
    chmod +x /usr/local/bin/yq

# =========================
# Install AWS CLI v2
# =========================
RUN curl -sL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.27.10.zip" -o awscliv2.zip && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf awscliv2.zip aws

# =========================
# Install Azure CLI
# =========================
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash && \
    rm -rf /var/lib/apt/lists/*

# =========================
# Install Google Cloud SDK
# =========================
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" \
    > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        google-cloud-cli \
        google-cloud-sdk-gke-gcloud-auth-plugin && \
    rm -rf /var/lib/apt/lists/*

# =========================
# Install kubectl
# =========================
RUN curl -L https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
    -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# =========================
# Install helm
# =========================
RUN curl -L https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz \
    -o helm.tar.gz && \
    tar -xzf helm.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf helm.tar.gz linux-amd64

# =========================
# Install clusterctl
# =========================
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/clusterctl-linux-amd64 \
    -o /usr/local/bin/clusterctl && \
    chmod +x /usr/local/bin/clusterctl

# =========================
# Prepare Cluster API local repository
# =========================
RUN mkdir -p \
    ${CAPI_REPO}/infrastructure-aws/${CAPA} \
    ${CAPI_REPO}/infrastructure-gcp/${CAPG} \
    ${CAPI_REPO}/infrastructure-azure/${CAPZ} \
    ${CAPI_REPO}/cluster-api/${CLUSTERCTL} \
    ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL} \
    ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL} \
    && echo "providers:" > /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: aws\n    url: ${CAPI_REPO}/infrastructure-aws/${CAPA}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: gcp\n    url: ${CAPI_REPO}/infrastructure-gcp/${CAPG}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: azure\n    url: ${CAPI_REPO}/infrastructure-azure/${CAPZ}/infrastructure-components.yaml\n    type: InfrastructureProvider" >> /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: kubeadm\n    url: ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/bootstrap-components.yaml\n    type: BootstrapProvider" >> /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: kubeadm\n    url: ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/control-plane-components.yaml\n    type: ControlPlaneProvider" >> /root/.cluster-api/clusterctl.yaml \
    && echo "  - name: cluster-api\n    url: ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/core-components.yaml\n    type: CoreProvider" >> /root/.cluster-api/clusterctl.yaml

# Download provider artifacts
RUN for i in metadata.yaml infrastructure-components.yaml; do \
      curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/${CAPA}/${i} -o ${CAPI_REPO}/infrastructure-aws/${CAPA}/${i} && \
      curl -L https://github.com/Stratio/cluster-api-provider-gcp/releases/download/${CAPG}/${i} -o ${CAPI_REPO}/infrastructure-gcp/${CAPG}/${i} && \
      curl -L https://github.com/kubernetes-sigs/cluster-api-provider-azure/releases/download/${CAPZ}/${i} -o ${CAPI_REPO}/infrastructure-azure/${CAPZ}/${i}; \
    done

# Create symlink for CAPG to support clusterctl's version normalization
# clusterctl expects 'v' prefix, but GitHub release doesn't use it
RUN ln -s ${CAPI_REPO}/infrastructure-gcp/${CAPG} ${CAPI_REPO}/infrastructure-gcp/v${CAPG}

RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/core-components.yaml -o ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/core-components.yaml && \
    curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/bootstrap-components.yaml -o ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/bootstrap-components.yaml && \
    curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/control-plane-components.yaml -o ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/control-plane-components.yaml && \
    curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/${CLUSTERCTL}/metadata.yaml -o ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml && \
    cp ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml ${CAPI_REPO}/bootstrap-kubeadm/${CLUSTERCTL}/metadata.yaml && \
    cp ${CAPI_REPO}/cluster-api/${CLUSTERCTL}/metadata.yaml ${CAPI_REPO}/control-plane-kubeadm/${CLUSTERCTL}/metadata.yaml

# =========================
# Working directory
# =========================
WORKDIR /upgrade

# =========================
# Aliases
# =========================
RUN echo 'alias k="kubectl"' >> ~/.bash_aliases && \
    echo 'alias capi-logs="kubectl -n capi-system logs -f deploy/capi-controller-manager"' >> ~/.bash_aliases && \
    echo 'alias capa-logs="kubectl -n capa-system logs -f deploy/capa-controller-manager"' >> ~/.bash_aliases && \
    echo 'alias capg-logs="kubectl -n capg-system logs -f deploy/capg-controller-manager"' >> ~/.bash_aliases && \
    echo 'alias capz-logs="kubectl -n capz-system logs -f deploy/capz-controller-manager"' >> ~/.bash_aliases && \
    echo 'alias kc-logs="kubectl -n kube-system logs -f deploy/keoscluster-controller-manager"' >> ~/.bash_aliases

# =========================
# Application files
# =========================
COPY ${UPGRADE_DIR}/templates /upgrade/templates
COPY ${UPGRADE_DIR}/upgrade-provisioner.py /upgrade
COPY ${UPGRADE_DIR}/ecr_pull_through.py /upgrade
COPY ${UPGRADE_DIR}/requirements.txt /upgrade

RUN pip3 install --no-cache-dir --break-system-packages -r requirements.txt

CMD ["/bin/bash"]
