FROM kindest/node:v1.24.6

# Tools versions
ENV CLUSTERCTL=v1.2.4
ENV CLUSTERAWSADM=v1.5.0
ENV HELM_VERSION=v3.11.0
ENV HELM_FILENAME=helm-${HELM_VERSION}-linux-amd64.tar.gz

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

RUN curl -L https://get.helm.sh/${HELM_FILENAME} -o /root/helm.tar.gz && tar -xf /root/helm.tar.gz -C /root && mv /root/linux-amd64/helm /usr/local/bin/helm && rm -rf /root/linux-amd64 && rm /root/helm.tar.gz \
    && chmod +x /usr/local/bin/helm && /usr/local/bin/helm repo add autoscaler https://kubernetes.github.io/autoscaler

RUN apt-get update && apt-get install -y unzip

RUN curl https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o awscliv2.zip \
    && unzip awscliv2.zip \
    && ./aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli \
    && rm -rf aws && rm awscliv2.zip
