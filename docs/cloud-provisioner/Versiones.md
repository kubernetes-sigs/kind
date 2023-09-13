# Actualización de versiones

> [kindest/node](https://hub.docker.com/r/kindest/node/tags)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.27.0 | 2023-04-11   | v1.28.0        | 2023-08-15          |

Files:   
*   pkg/apis/config/defaults/image.go  
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile

> [clusterctl](https://github.com/kubernetes-sigs/cluster-api/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.5.1  | 2023-08-29   | v1.5.1         | 2023-08-29          |

Files:   
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile

> [clusterawsadm](https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v2.2.1  | 2023-07-20   | v2.2.1         | 2022-07-20          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile

> [pause](https://github.com/kubernetes/kubernetes/blob/master/build/pause/CHANGELOG.md)
| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v.3.9   | 2023-06-16   | v.3.9          | 2023-06-16          |

Files:
*   DEPENDENCIES

> [Helm](https://github.com/helm/helm/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v3.11.3 | 2023-08-10   | v3.12.3        | 2023-08-10          |

Files:  
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile

> [cluster_auto_scaler](https://github.com/kubernetes/autoscaler/releases) 

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v9.29.1 (chart) | 2022-06-15   | v9.29.3        | 2023-08-29          |
| v1.27.2 (cluster-autoscaler) | 2023-05-29   | v1.28.0        | 2023-08-31          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile

> [Tigera_operator](https://github.com/projectcalico/calico/releases) (https://github.com/tigera/operator/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.30.5 (tigera) | 2023-08-05   | v1.31.0        | 2023-09-01          |
| v3.26.1 (calicoctl)(Tigera chart) | 2022-06-17   | v3.26.1        | 2022-06-17          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile
*   pkg/cluster/internal/create/actions/createworker/templates/common/calico-helm-values.tmpl

> [aws-ebs-csi-driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver/releases) (eksctl utils describe-addon-versions --kubernetes-version 1.26 --name aws-ebs-csi-driver | grep AddonVersion)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.19.0-eksbuild.2 (EKS)  |  2023-05-16  |   v1.22.0-eksbuild.2       | 2023-08-16          |
| v2.20.0 (AWS unmanaged)| 2023-06-20     | v2.22.0         | 2023-08-16          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile
*   controllers/templates/aws/aws.eks.tmpl

> [coredns](eksctl utils describe-addon-versions --kubernetes-version 1.26 --name coredns | grep AddonVersion)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.9.3-eksbuild.3 (EKS)  |    |   v1.9.3-eksbuild.3       |           |

Files:  
*   DEPENDENCIES
*   controllers/templates/aws/aws.eks.tmpl

> [kube-proxy](eksctl utils describe-addon-versions --kubernetes-version 1.26 --name kube-proxy | grep AddonVersion)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.24.15-eksbuild.2 (EKS)  |    |   v1.26.7-eksbuild.2       |           |

Files:  
*   DEPENDENCIES
*   controllers/templates/aws/aws.eks.tmpl

> [vpc-cni](https://github.com/aws/amazon-vpc-cni-k8s/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.12.6-eksbuild.2  | 2023-03-20   | v1.14.1-eksbuild.1         | 2023-09-08          |

Files:  
*   DEPENDENCIES
*   controllers/templates/aws/aws.eks.tmpl

> [cluster-api-aws / cluster-api-aws-templates](https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v2.2.1  | 2023-07-20   | v2.2.1         | 2022-07-20          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/create/actions/createworker/aws.go

> [cluster-api-gcp / cluster-api-gcp-templates](https://github.com/kubernetes-sigs/cluster-api-provider-gcp/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.4.0  | 2023-07-17   | v1.4.0         | 2023-07-17          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/create/actions/createworker/gcp.go

> [cluster-api-azure / cluster-api-azure-templates](https://github.com/kubernetes-sigs/cluster-api-provider-azure/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.10.3 (AKS) | 2023-09-06   | v1.10.3        | 2023-09-06          |
| v1.9.8 (Azure unmanaged) | 2023-09-06   | v1.9.8         | 2023-09-06          |

Files:
*   DEPENDENCIES
*   pkg/cluster/internal/providers/docker/stratio/Dockerfile
*   pkg/cluster/internal/create/actions/createworker/azure.go

> [external-attacher](https://github.com/kubernetes-csi/external-attacher/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v4.3.0-eks-1-27-3 | 2023-04-28   | v4.3.0-eks-1-27-3         | 2023-04-28          |

Files:
*   DEPENDENCIES

> [external-provisioner](https://github.com/kubernetes-csi/external-provisioner/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v3.5.0-eks-1-27-3 | 2023-04-28   | v3.5.0-eks-1-27-3         | 2023-04-28          |

Files:
*   DEPENDENCIES

> [external-resizer](https://github.com/kubernetes-csi/external-resizer/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.8.0-eks-1-27-3 | 2023-09-11   | v1.8.0-eks-1-27-3         | 2023-09-11          |

Files:
*   DEPENDENCIES

> [livenessprobe](https://github.com/kubernetes-csi/livenessprobe/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v2.10.0-eks-1-27-3 | 2023-04-28   | v2.5.0-eks-1-27-3         | 2023-04-28          |
| v2.8.0 (Azure unmanaged) | 2023-04-28   | v2.10.0         | 2023-08-28          |

> [gcp-compute-persistent-disk-csi-driver](https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.7.1  | 2022-10-21   | v1.9.2         | 2022-03-17          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/create/actions/createworker/files/gcp/gcp-compute-persistent-disk-csi-driver.yaml

> [csi-node-driver-registrar](https://github.com/kubernetes-csi/node-driver-registrar/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v2.8.0-eks-1-27-3  | 2023-04-28   | v2.8.0-eks-1-27-3         | 2023-02-28          |
| v2.7.0 (GCP) | 2022-12-28  | v2.8.0         | 2023-04-28          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/create/actions/createworker/files/gpc/gcp-compute-persistent-disk-csi-driver.yaml

> [volume-modifier-for-k8s](https://gallery.ecr.aws/ebs-csi-driver/volume-modifier-for-k8s)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v0.1.0  | 2023-08-14   | v.0.1.2        | 2023-08-14          |

Files:
*   DEPENDENCIES

> [csi-snapshotter](https://github.com/kubernetes-csi/external-snapshotter/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v6.2.1-eks-1-27-3  | 2023-04-01   | v6.2.2         | 2023-05-27          |

Files:  
*   DEPENDENCIES
*   pkg/cluster/internal/create/actions/createworker/files/gcp/gcp-compute-persistent-disk-csi-driver.yaml

> [csi-resizer](https://github.com/kubernetes-csi/external-resizer/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.4.0  | 2022-01-21   | v1.7.0         | 2022-12-28          |

Files:  
*  DEPENDENCIES
*  pkg/cluster/internal/create/actions/createworker/files/gcp/gcp-compute-persistent-disk-csi-driver.yaml

> [csi-attacher](https://github.com/kubernetes-csi/external-attacher/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v4.0.0 (Azure unmanaged) | 2023-09-13   | v4.3.0         | 2023-04-28          |
| v4.2.0 (GCP) | 2023-02-01   | v4.3.0         | 2023-04-28          |

Files:  
*  DEPENDENCIES
*  pkg/cluster/internal/create/actions/createworker/files/gcp/gcp-compute-persistent-disk-csi-driver.yaml

> [csi-provisioner](https://github.com/kubernetes-csi/external-provisioner/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v3.3.0 (Azure unmanaged) | 2022-09-27   | v3.5.0         | 2023-04-28          |
| v3.4.0 (GCP) | 2022-12-28   | v3.5.0         | 2023-04-28          |

Files:  
*  DEPENDENCIES
*  pkg/cluster/internal/create/actions/createworker/files/gcp/gcp-compute-persistent-disk-csi-driver.yaml

> [azure-cloud-controller-manager](https://github.com/kubernetes-sigs/cloud-provider-azure/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v.1.24.18 | 2023-04-21 | v.1.28.0       | 2023-08-22          |

files:
*  DEPENDENCIES

> [azure-cloud-node-manager](https://github.com/kubernetes-sigs/cloud-provider-azure/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v.1.24.18 | 2023-04-21 | v.1.28.0       | 2023-08-22          |

files:
*  DEPENDENCIES

> [azuredisk-csi-driver / azuredisk-csi-driver-chart](https://github.com/kubernetes-sigs/azuredisk-csi-driver/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.27.1  | 2023-04-31   | v1.28.3         | 2023-08-25          |

Files:
*  DEPENDENCIES

> [azurefile-csi-driver / azurefile-csi-driver-chart](https://github.com/kubernetes-sigs/azurefile-csi-driver/releases)

| Version | Release Date | Latest Version | Latest Release Date |
| ------- | ------------ | -------------- | ------------------- |
| v1.27.1 | 2023-07-20  | v1.28.4         | 2023-09-08          |

Files:
*  DEPENDENCIES
