# Changelog

All notable changes to this project will be documented in this file.

## 0.9.0 (upcoming)

* [PLT-4247] Add Kubernetes 1.34 and 1.35 support: EKS (addons, coredns, kube-proxy), Azure VMs (cloud-provider-azure 1.35.3, azuredisk 1.34.4, azurefile 1.35.3, cluster-autoscaler 9.57.0), GKE (coredns v1.13.2); bootstrap cluster updated to kindest/node:v1.35.5
* [PLT-4266] Bump Calico v3.30.2→v3.31.5, tigera-operator controller v1.38.5→v1.40.11, FluxCD chart 2.17.2→2.18.4 (all providers, all supported minors)
* [PLT-4238] Make Calico Whisker and Goldmane observability components optional via `calico.observability_enabled` descriptor field (default: disabled)
* [PLT-4154] - Fix vulnerabilities: CAPI v1.10.10, CAPA v2.9.3, CAPZ v1.21.3, Calico v3.31.5, cert-manager v1.20.2, FluxCD 2.17.2
* [PLT-4161] - Add ecr_pull_through.py migration script and ECR pull-through support in upgrade-provisioner
* [PLT-3963] - Adaptar cloud-provisioner a Semantic Versioning
* [PLT-3091] - Adaptar cloud-provisioner para soportar ECR central: soportar prefijos
* [Fix] Fix clusterawsadm YAML corruption when writing eks.config and add missing IAM permissions (PutRolePolicy, DeleteRolePolicy, GetRolePolicy, ListRolePolicies) to stratio-aws-temp-policy
* [PLT-4154] Downgrade Calico v3.31.5→v3.30.2 and FluxCD flux-cli v2.8.7→v2.7.5 for compatibility
* [PLT-4154] Fix vulnerabilities: CAPI v1.10.10, CAPA v2.9.3, CAPZ v1.21.3, Calico v3.31.5, cert-manager v1.20.2, FluxCD 2.17.2 (flux-cli v2.7.5), cluster-operator 0.6.2
* [PLT-4161] Backport ecr_pull_through_cache_enabled to 0.17.0-0.8.5

## 0.17.0-0.8.4 (2026-03-25)

* [Fix]  Azure image to version V2, GKE coredns version to 1.11.3, Add whisker and goldmane images to list

## 0.17.0-0.8.3 (2026-03-20)

* [PLT-3877] -  cloud-provisioner upgrade,  update cluster-operator version

## 0.17.0-0.8.2 (2026-03-19)

* [PLT-3877] -  [Azure/EKS/GKE] Cloud-Provisioner upgrade

## 0.17.0-0.8.1 (2026-03-13)

* [PLT-3691] -  [Fix] Flux2 chart upgrading provisioner

## Previous development

### Branched to branch-0.8 (2026-02-20)

* [PLT-3794] -  Feature: Add gitops-enabled in keoscluster to avoid FluxCD deployments and asssociated helmreleases  - [`#876`](https://github.com/Stratio/kind/pull/876)
* [PLT-3770] -  Subir Flux a 2.7.5  - [`#873`](https://github.com/Stratio/kind/pull/873)
* [PLT-3777] -  Feature: Adaptate to Semantic Versioning  - [`#875`](https://github.com/Stratio/kind/pull/875)
* [PLT-3697] -  [EKS] Fallo al desplegar "AWS LB controller"  - [`#871`](https://github.com/Stratio/kind/pull/871)
* [PLT-1548] -  [GKE] Activar Workload Identity  - [`#867`](https://github.com/Stratio/kind/pull/867)
* [PLT-3524] -  [GKE] Gestion metadata.labesl en los node-pools  - [`#866`](https://github.com/Stratio/kind/pull/866)
* [PLT-3368] -  Bump cloud-node/controller-manager to version 1.34.2  - [`#864`](https://github.com/Stratio/kind/pull/864)
* [PLT-3365] -  Bump CAPZ to version 1.21.1 & azureserviceoperator to 2.11.0  - [`#863`](https://github.com/Stratio/kind/pull/863)
* [PLT-3360] -  Solucionar vulnerabilidades en las imágenes de cert-manager (cluster-api)  - [`#859`](https://github.com/Stratio/kind/pull/859)
* [PLT-3362] -  bump azurefile csi to version v1.34.1  - [`#862`](https://github.com/Stratio/kind/pull/862)
* [PLT-3363] -  bump azuredisk csi to version v1.33.5  - [`#861`](https://github.com/Stratio/kind/pull/861)
* [PLT-3427] -  [Fix] Condición de Carrera en cloud-provisioner Causa Fallo al Obtener el Kubeconfig del Clúster  - [`#860`](https://github.com/Stratio/kind/pull/860)
* [PLT-3359] -  Bump aws-load-balancer-controller to version 1.14.1  - [`#855`](https://github.com/Stratio/kind/pull/855)
* [PLT-2587] -  [Clouds] Solucionar vulnerabilidades en cloud-provisioner 0.7  - [`#853`](https://github.com/Stratio/kind/pull/853)
* [PLT-3376] -  Bump Flux components  - [`#852`](https://github.com/Stratio/kind/pull/852)
* [PLT-3038] -  Adaptar estructura para soportar multitenant  - [`#847`](https://github.com/Stratio/kind/pull/847)
* [PLT-2887] -  Configuración de cifrado por defecto para EBS  - [`#846`](https://github.com/Stratio/kind/pull/846)
* [PLT-3024] -  Remove use of kube-rbac-proxy form `cluster-operator`  - [`#845`](https://github.com/Stratio/kind/pull/845)
* [PLT-3080] -  Adaptar cloud-provisioner para soportar ECR central  - [`#840`](https://github.com/Stratio/kind/pull/840)
* [PLT-2591] -  Bump CAPZ version to v1.21.0  - [`#839`](https://github.com/Stratio/kind/pull/839)
* [PLT-2587] -  Fix cloud-provisioner vulnerabilities & update components  - [`#838`](https://github.com/Stratio/kind/pull/838)
* [PLT-2953] -  Fix renewal credentials docs  - [`#834`](https://github.com/Stratio/kind/pull/834)
* [PLT-2655] -  Update kube-rbac-proxy repository  - [`#837`](https://github.com/Stratio/kind/pull/837)
* [PLT-2998] -  [Azure] Fix CSI deployment with public repositories  - [`#836`](https://github.com/Stratio/kind/pull/836)
* [PLT-2655] -  Fix cluster-operator & kube-rbac-proxy vulnerabilities  - [`#835`](https://github.com/Stratio/kind/pull/835)
* [PLT-2660] -  Bump CSI Azure to version 1.33.4  - [`#832`](https://github.com/Stratio/kind/pull/832)
* [PLT-2723] -  Bump aws-ebs-csi-driver, coredns, kube-proxy and vpc-cni addons versions  - [`#829`](https://github.com/Stratio/kind/pull/829)
* [PLT-2653] -  Bump cluster-autoscaler version to v1.33.0  - [`#828`](https://github.com/Stratio/kind/pull/828)
* [PLT-2643] -  Bump aws-load-balancer controller version to v2.13.4  - [`#827`](https://github.com/Stratio/kind/pull/827)
* [PLT-2656] -  Bump Tigera Operator version to v3.30.2  - [`#825`](https://github.com/Stratio/kind/pull/825)
* [PLT-2636] -  [Clouds] Solucionar vulnerabilidades en (cluster-api-controller:v1.7.4) ClusterAPI  - [`#826`](https://github.com/Stratio/kind/pull/826)
* [PLT-2664] -  Bump Flux components  - [`#824`](https://github.com/Stratio/kind/pull/824)
* [PLT-2634] -  Bump CAPA version to v2.8.4  - [`#820`](https://github.com/Stratio/kind/pull/820)
* [PLT-2583] -  Clean dependencies to improve vulnerabilities management  - [`#817`](https://github.com/Stratio/kind/pull/817)
* [PLT-2562] -  Fix  image reference during cluster creation  - [`#815`](https://github.com/Stratio/kind/pull/815)
* [PLT-2389] -  Fix  image reference during cluster creation  - [`#808`](https://github.com/Stratio/kind/pull/808)
* [PLT-2379] -  Fix namespace reference  - [`#804`](https://github.com/Stratio/kind/pull/804)
* [PLT-2335] -  Bump cluster-operator to 0.5.2 version  - [`#805`](https://github.com/Stratio/kind/pull/805)
* [PLT-2335] -  No levanta docker keos-installer con parámetro role_arn:'false'  - [`#800`](https://github.com/Stratio/kind/pull/800)
* [PLT-2108] -  Configurar Assume role (STS) de forma manual  - [`#785`](https://github.com/Stratio/kind/pull/785)
* [PLT-2259] -  Revisar la creación del kubeconfig para EKS durante la instalación con cloud-provisioner  - [`#796`](https://github.com/Stratio/kind/pull/796)
* [PLT-1549] -  Activar NodePool SecureBoot  - [`#748`](https://github.com/Stratio/kind/pull/748)
* [PLT-1762] -  [EKS] Soportar instalaciones con Assume Role  - [`#756`](https://github.com/Stratio/kind/pull/756)
* [PLT-2226] -  [cloud-provisioner] Usar repo privado por defecto  - [`#794`](https://github.com/Stratio/kind/pull/794)
* [PLT-2289] -  Add safe-to-evict annotations in Flux pods  - [`#795`](https://github.com/Stratio/kind/pull/795)
* [PLT-2244] -  Bump cluster-operator to 0.5.1 version to upgrade issues  - [`#783`](https://github.com/Stratio/kind/pull/783)
* [PLT-2204] -  Ensure referencing cloud-provisioner image release instead of prerelease version when creating a cluster  - [`#787`](https://github.com/Stratio/kind/pull/787)
* [PLT-2244] -  Deshabilitar CriVolume por defecto en instalación y en operación del cluster  - [`#789`](https://github.com/Stratio/kind/pull/789)
* [PLT-1496] -  Ensure CAPG provisioner version references are set to 1.6.1-0.3.1  - [`#788`](https://github.com/Stratio/kind/pull/788)
* [PLT-2099] -  [Upgrade] Problemas actualización Add-On CoreDNS en actualización de EKS  - [`#777`](https://github.com/Stratio/kind/pull/777)
* [PLT-2139] -  Fix cloud-provisioner upgrade issue when retrying upgrade …  - [`#780`](https://github.com/Stratio/kind/pull/780)
* [PLT-2124] -  Bump cluster-autoscaler to v1.32.0 version and its chart version to 9.46.6  - [`#776`](https://github.com/Stratio/kind/pull/776)
* [PLT-2098] -  Improve kubernetes version checks during cloud-provisioner-upgrade  - [`#772`](https://github.com/Stratio/kind/pull/772)
* [PLT-2052] -  Fix Azure charts versions references  - [`#768`](https://github.com/Stratio/kind/pull/768)
* [PLT-1910] -  Instalar en Azure con infra creada  - [`#758`](https://github.com/Stratio/kind/pull/758)
* [PLT-1958] -  Improve …  - [`#752`](https://github.com/Stratio/kind/pull/752)
* [PLT-1823] -  Improve Clouds credentials management documentation (#707)  - [`#736`](https://github.com/Stratio/kind/pull/736)
* [PLT-1849] -  Fix aws-load-balancer-controller annotation  - [`#732`](https://github.com/Stratio/kind/pull/732)
* [PLT-1652] -  Allow skipping kubernetes intermediate version during upgrade  - [`#718`](https://github.com/Stratio/kind/pull/718)
* [PLT-1849] -  Error 403 sts:AssumeRoleWithWebIdentity en ingress-nginx-controller  - [`#720`](https://github.com/Stratio/kind/pull/720)
* [PLT-1887] -  Dynamic region describe  - [`#714`](https://github.com/Stratio/kind/pull/714)
* [PLT-1621] -  Add kubernetes 1.32 support  - [`#689`](https://github.com/Stratio/kind/pull/689)
* [PLT-1741] -  Bump cluster-operator references to 0.5.0 version. Update EKS addons dependencies documentation  - [`#701`](https://github.com/Stratio/kind/pull/701)
* [PLT-1682] -  Improve kindest/node and stratio-capi-image management  - [`#685`](https://github.com/Stratio/kind/pull/685)
* [PLT-1317] -  Remove non-suported AKS, managed AWS and managed GCP references  - [`#692`](https://github.com/Stratio/kind/pull/692)
* [PLT-1628] -  Fix capz images registry and repository references. Replace cloud-provider-azure …  - [`#686`](https://github.com/Stratio/kind/pull/686)
* [PLT-1394] -  Bump Flux version to 2.14.1  - [`#662`](https://github.com/Stratio/kind/pull/662)
* [PLT-1393] -  Bump Tigera Operator version to v3.29.1  - [`#661`](https://github.com/Stratio/kind/pull/661)
* [PLT-1628] -  Fix coredns, cluster-api-gcp and kube-rbac-proxy image registry and repository references  - [`#675`](https://github.com/Stratio/kind/pull/675)
* [PLT-1332] -  [GKE] Validaciones parámetros GKE  - [`#657`](https://github.com/Stratio/kind/pull/657)
* [PLT-1330] -  CMEK, SA & CIDRs  - [`#642`](https://github.com/Stratio/kind/pull/642)
* [PLT-964] -  Validaciones nuevos parámetros GKE  - [`#626`](https://github.com/Stratio/kind/pull/626)
* [PLT-1156] -  Add deny-all-egress-imds_gnetpol  - [`#629`](https://github.com/Stratio/kind/pull/629)
* [PLT-1309] -  Update docker images requirements documentation. Include stratio-capi-image to cicd flow  - [`#663`](https://github.com/Stratio/kind/pull/663)
* [PLT-719] -  Doc 0.5 to master  - [`#645`](https://github.com/Stratio/kind/pull/645)
* [PLT-1178] -  fix aws-load-balancer-controller  - [`#640`](https://github.com/Stratio/kind/pull/640)




### Branched to branch-0.17.0-0.6 (2024-10-25)

* [Core] Ensure CoreDNS replicas are assigned to different nodes
* [Core] Added the default creation of volumes for containerd, etcd and root, if not indicated in the keoscluster
* [Core] Support k8s v1.30
* [Core] Deprecated Kubernetes versions prior to 1.28
* [PLT-817] Bump Tigera Operator version to v3.28.2
* [PLT-965] Disable managed Monitoring and Logging
* [PLT-806] Support for private clusters on GKE
* [PLT-920] Added use-local-stratio-image flag to reuse local image
* [PLT-917] Replace coredns yaml files with a single coredns tmpl file
* [PLT-929] Removed calico installation as policy manager by helm chart in GKE
* [PLT-911] Support for Disable External Endpoint in GKE
* [PLT-923] Remove path /stratio from container image reference for kube-rbac-proxy image
* [PLT-992] Uncouple CAPX from cloud provisioner and allow to specify versions in clusterconfig
* [PLT-988] Uncouple CAPX from Dockerfile
* [PLT-964] Add GKE Private Cluster Validations
