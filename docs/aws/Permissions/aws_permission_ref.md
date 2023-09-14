# AWS Permissions

> user: sa-XXXXX  
> permissions: janr-policy  

### Permissions Table

**Test:** cloud-provisioner create cluster --name eks-cl01 --retain --vault-password 123456 --avoid-creation

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| ec2:DescribeAvailabilityZones | failed to get AZs | DescribeAvailabilityZones | * | cloud-provisioner |
| ecr:GetAuthorizationToken | Not Authorized to perform: ecr:GetAuthorizationToken on resource: * | GetAuthorizationToken | * | cloud-provisioner |
| ecr:GetDownloadUrlForLayer | Image pull | GetDownloadUrlForLayer | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |
| ecr:BatchGetImage | Image pull | BatchGetImage | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |
| ecr:BatchCheckLayerAvailability | Image pull | BatchCheckLayerAvailability | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |

**Test:** cloud-provisioner create cluster --name eks-cl01 --retain --vault-password 123456 --keep-mgmt

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| ec2:CreateVpc | "action\":\"ec2:CreateVpc | CreateVpc | arn:aws:ec2:*:268367799918:vpc/*  | cloud-provisioner |
| ec2:CreateTags | "action\":\"ec2:CreateTags | CreateTags | * | cloud-provisioner |
| ec2:DescribeVpcs | VPC resource is missing in AWS; failed to describe VPC resources | DescribeVpcs | * | cloud-provisioner |
| ec2:DescribeVpcAttribute | failed to describe VPC resources | DescribeVpcAttribute | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |
| ec2:ModifyVpcAttribute |  failed to set enableDnsHostnames vpc attribute | ModifyVpcAttribute | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |
| ec2:ModifyVpcAttribute | failed to describe subnets in vpc | ModifyVpcAttribute | arn:aws:ec2:*:268367799918:vpc/* | cloud-provisioner |
| ec2:DescribeSubnets | failed to describe subnets in vpc | DescribeSubnets | * | cloud-provisioner |
| ec2:DescribeRouteTables  | failed to describe route tables in vpc | DescribeRouteTables | * | cloud-provisioner |
| ec2:DescribeNatGateways | failed to describe nat gateways in vpc | DescribeNatGateways | * | cloud-provisioner |
| ec2:CreateSubnet | failed to create subnet | CreateSubnet | * | cloud-provisioner |
| ec2:DescribeInternetGateways | failed to describe internet gateways in vpc | DescribeInternetGateways | * | cloud-provisioner |
| ec2:CreateInternetGateway | failed to create internet gateway | CreateInternetGateway | arn:aws:ec2:*:268367799918:internet-gateway/* | cloud-provisioner |
| ec2:ModifySubnetAttribute | failed to modify subnet attribute | ModifySubnetAttribute | * | cloud-provisioner |
| ec2:AttachInternetGateway | failed to attach internet gateway | AttachInternetGateway | * | cloud-provisioner |
| ec2:AllocateAddress | failed to create one or more IP addresses for NAT gateways | AllocateAddress | * | cloud-provisioner |
| ec2:DescribeAddresses | failed to query addresses | DescribeAddresses | * | cloud-provisioner |
| ec2:CreateRouteTable | failed to create route table | CreateRouteTable | arn:aws:ec2:*:268367799918:route-table/* | cloud-provisioner |
| ec2:AssociateRouteTable | failed to associate route table | AssociateRouteTable | * | cloud-provisioner |
| ec2:CreateNatGateway | failed to create nat gateway | CreateNatGateway | * | cloud-provisioner |
| ec2:DescribeSecurityGroups | failed to describe security groups | DescribeSecurityGroups | * | cloud-provisioner |
| ec2:ReplaceRoute | Failed to replace outdated route on managed RouteTable | ReplaceRoute | arn:aws:ec2:*:268367799918:route-table/* | cloud-provisioner |
| ec2:CreateSecurityGroup | failed to create security group | CreateSecurityGroup | * | cloud-provisioner |
| ec2:DescribeInstances | failed to describe bastion host | DescribeInstances | * | cloud-provisioner |
| iam:GetRole | failed to get role | GetRole | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:ListAttachedRolePolicies | error listing role polices | ListAttachedRolePolicies | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| eks:DescribeCluster | failed to describe eks cluster | DescribeCluster | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| eks:CreateCluster | failed to create eks cluster | CreateCluster | * | cloud-provisioner |
| eks:TagResource | failed to tag eks cluster | TagResource | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| iam:CreateRole | failed to create role | CreateRole | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:TagRole | not authorized to perform: iam:TagRole on resource: arn:aws:iam::268367799918:role/eks-cl01-iam-service-role | TagRole | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:GetPolicy | error getting policy arn:aws:iam::aws:policy/AmazonEKSClusterPolicy: AccessDenied | GetPolicy | * | cloud-provisioner |
| iam:AttachRolePolicy | not authorized to perform: iam:AttachRolePolicy on resource: role eks-cl01-iam-service-role | AttachRolePolicy | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:PassRole | failed to pass role | PassRole | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:ListOpenIDConnectProviders | failed to list open id connect providers | ListOpenIDConnectProviders | * | cloud-provisioner |
| secretsmanager:CreateSecret | not authorized to perform: secretsmanager:TagResource | CreateSecret | arn:aws:secretsmanager:*:268367799918:secret:aws.cluster.x-k8s.io/* | cloud-provisioner |
| iam:GetOpenIDConnectProvider | error getting provider | GetOpenIDConnectProvider | arn:aws:iam::268367799918:oidc-provider/* | cloud-provisioner |
| secretsmanager:DeleteSecret | not authorized to perform: secretsmanager:DeleteSecret | DeleteSecret | arn:aws:secretsmanager:*:268367799918:secret:aws.cluster.x-k8s.io/* | cloud-provisioner |
| iam:CreateOpenIDConnectProvider | error creating provider | CreateOpenIDConnectProvider | arn:aws:iam::268367799918:oidc-provider/* | cloud-provisioner |
| ssm:GetParameter | failed to get ami SSM parameter | GetParameter | arn:*:ssm:*:*:parameter/aws/service/eks/optimized-ami/* | cloud-provisioner |
| eks:ListAddons | failed to list eks addons | ListAddons | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| eks:DescribeAddon | failed to describe eks addons | DescribeAddonVersions | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| eks:CreateAddon | failed to create eks addon | CreateAddon | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| eks:DescribeAddonVersions | failed reconciling eks addons: getting installed eks addons | DescribeAddonVersions | * | cloud-provisioner |
| ec2:DescribeImages | failed to get root volume | DescribeImages | * | cloud-provisioner |
| ec2:RunInstances | Not authorized to perform: ec2:RunInstances | RunInstances | * | cloud-provisioner |
| ec2:AssociateVpcCidrBlock | failed to associate vpc cidr block | AssociateVpcCidrBlock | * | cloud-provisioner |
| ec2:DissociateVpcCidrBlock | failed to dissociate vpc cidr block | DissociateVpcCidrBlock | * | cloud-provisioner |

**Test:** cloud-provisioner create cluster --name eks-cl01 --retain --vault-password 123456 (same permissions as --keep-mgmt)

**Test:** clusterctl move --kubeconfig /home/jnovoa/.kube/config --to-kubeconfig local_kubeconfig --namespace cluster-eks-cl01 --dry-run
Performing move...
********************************************************
This is a dry-run move, will not perform any real action
********************************************************
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ clusterctl move --kubeconfig /home/jnovoa/.kube/config --to-kubeconfig local_kubeconfig --namespace cluster-eks-cl01 (no needed additonal permissions)  
Performing move...  
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ clusterctl move --to-kubeconfig /home/jnovoa/.kube/config --kubeconfig local_kubeconfig --namespace cluster-eks-cl01
Performing move...  
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ clusterctl --kubeconfig /home/jnovoa/.kube/config describe cluster eks-cl01 -n cluster-eks-cl01  
NAME                                                            READY  SEVERITY  REASON  SINCE  MESSAGE  
Cluster/eks-cl01                                                True                     90s  
├─ClusterInfrastructure - AWSManagedCluster/eks-cl01  
├─ControlPlane - AWSManagedControlPlane/eks-cl01-control-plane  True                     86s  
└─Workers  
  ├─MachineDeployment/eks-cl01-worker-1-md-0                    True                     101s  
  │ └─Machine/eks-cl01-worker-1-md-0-8cc9d5b8cx56zgn-4gr5k      True                     79s  
  ├─MachineDeployment/eks-cl01-worker-1-md-1                    True                     100s  
  │ └─Machine/eks-cl01-worker-1-md-1-546ff48c8dxpzswd-98gmg     True                     80s  
  └─MachineDeployment/eks-cl01-worker-1-md-2                    True                     100s  
    └─Machine/eks-cl01-worker-1-md-2-859dc7f565x5f98d-tls2s     True                     79s  

**Test:** Auto-Scale (up/down) (deploy "n" nginx pods and check if the nodes are scaled up)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 6 | 1 | 6 | 5 | ScalingUp | 3m53s | v1.24.11 | 
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | 0 | Running   | 3m50s | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 2 | 1 | 2 | 1 | ScalingUp | 3m51s | v1.24.11 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 6 | 6 | 6 | |0 | Running | 6m27s | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 6m24s | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 2 | 2 | 2 | |0 | Running | 6m25s | v1.24.11 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 6 | 1 | 6 | 5 | ScalingDown | 19m | v1.24.11 | 
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | 0 | Running   | 19m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 2 | 1 | 2 | 1 | ScalingDown | 19m | v1.24.11 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| ec2:TerminateInstances | failed to terminate instance | TerminateInstances | * | cloud-provisioner |

**Test:** Scale Manually (up/dpwn) kubectl -n cluster-eks-cl01 scale --replicas 3 machinedeployments --all (same as above)

**Test:** Create new MachineDeployment Manually (same as above) (scale ok)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-3 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 48s | v1.24.11 |

**Test:** Destroy Machine on AWS UI (self-healing) (eks-cl01-worker-1-md-2) (same as above)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-eks-cl01 | eks-cl01-worker-1-md-0 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-2 | eks-cl01 | 1 | 1 | 1 | |0 |  ScalingUp| 64m | v1.24.11 |
| cluster-eks-cl01 | eks-cl01-worker-1-md-3 | eks-cl01 | 1 | 1 | 1 | |0 | Running | 48s | v1.24.11 |

**Test:** kubectl --kubeconfig /home/jnovoa/.kube/config -n cluster-eks-cl01 patch awsmanagedcontrolplanes.controlplane.cluster.x-k8s.io eks-cl01-control-plane --type merge -p '{"spec":{"version":"v1.25.0"}}'  
(upgrade control-plane k8s version from 1.24 to 1.25) 

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| eks:UpdateClusterVersion | failed to update cluster version | UpdateClusterVersion | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |

|   Cluster name | Status | Kubernetes version| Provider |
| --- | --- | --- | --- |
| eks-cl01 |	Updating | 1.24 | EKS |
| eks-cl01 |    Active | 1.25 Update now | EKS |

**Test:** kubectl --kubeconfig /home/jnovoa/.kube/config -n cluster-eks-cl01 patch machinedeployments.cluster.x-k8s.io eks-cl01-worker-1-md-0 --type merge -p '{"spec":{"template":{"spec":{"version":"v1.25.0"}}}}'  (no more permissions needed)  

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eks-cl01-worker-1-md-0 |  eks-cl01 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.11  |
| eks-cl01-worker-1-md-1 |  eks-cl01 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.11  |
| eks-cl01-worker-1-md-2 |  eks-cl01 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.11  |

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eks-cl01-worker-1-md-0 | eks-cl01 | 2 | 2 | 1 | 0 | ScalingDown | 18m | v1.25.0  |
| eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | 0 | Running     | 18m | v1.24.11 |
| eks-cl01-worker-1-md-2 | eks-cl01 | 1 | 1 | 1 | 0 | Running     | 18m | v1.24.11 |

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eks-cl01-worker-1-md-0 | eks-cl01 | 1 | 1 | 1 | 0 | Running | 21m | v1.25.0  |
| eks-cl01-worker-1-md-1 | eks-cl01 | 1 | 1 | 1 | 0 | Running | 21m | v1.25.0 | 
| eks-cl01-worker-1-md-2 | eks-cl01 | 1 | 1 | 1 | 0 | Running | 21m | v1.25.0 | 


**Test:** Delete cluster (From local container)
kubectl --kubeconfig local_kubeconfig -n cluster-eks-cl01 delete cluster eks-cl01

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| eks:DeleteCluster | not authorized to perform: eks:DeleteCluster on resource arn:aws:eks:eu-west-1:268367799918:cluster/eks-cl01 | DeleteCluster | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |
| iam:DetachRolePolicy | not authorized to perform: iam:DetachRolePolicy on resource: role eks-cl01-iam-service-role | DetachRolePolicy | arn:aws:iam::268367799918:role/* | cloud-provisioner |
| iam:DeleteOpenIDConnectProvider | not authorized to perform: iam:DeleteOpenIDConnectProvider on resource: arn:aws:iam::268367799918:oidc-provider/oidc.eks.eu-west-1.amazonaws.com/id/B6F3AF0562AECA0EE0775E9FC8AE08DC | DeleteOpenIDConnectProvider | arn:aws:iam::268367799918:oidc-provider/* | cloud-provisioner |
| ec2:DeleteSecurityGroup | failed to delete security group | DeleteSecurityGroup | * | cloud-provisioner | 
| ec2:DisassociateRouteTable | failed to disassociate route table "rtb-0efe6491b57f51e10" from subnet | DisassociateRouteTable | * | cloud-provisioner |
| ec2:DeleteRouteTable | failed to delete route table "rtb-0efe6491b57f51e10" | DeleteRouteTable | * | cloud-provisioner |
| ec2:DeleteNatGateway | failed to delete NAT gateway "nat-0a0b0c0d0e0f0a0b0" | DeleteNatGateway | * | cloud-provisioner |
| ec2:ReleaseAddress | failed to release ElasticIP "eipalloc-0b9b95649ebe9fe11": UnauthorizedOperation: | ReleaseAddress | * | cloud-provisioner |
| ec2:DetachInternetGateway | failed to detach internet gateway "igw-027278b62139ce652": UnauthorizedOperation: You are not authorized | DetachInternetGateway | * | cloud-provisioner |
| ec2:DeleteSubnet |failed to delete subnet "subnet-0c5ff226c483b3818": UnauthorizedOperation | DeleteSubnet | * | cloud-provisioner |
| ec2:DeleteInternetGateway | failed to delete internet gateway "igw-027278b62139ce652": UnauthorizedOperation: You are not authorized | DeleteInternetGateway | * | cloud-provisioner |

**Test**: Keos Install

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| iam:ListInstanceProfilesForRole | not authorized to perform: iam:ListInstanceProfilesForRole on resource: role eks-iam-for-sa-external-dns-role-eks-cl01 | ListInstanceProfilesForRole | arn:aws:iam::268367799918:role/* | keos |
| iam:CreateInstanceProfile | Unable to create instance profile for role eks-iam-for-sa-external-dns-role-eks-cl01: An error occurred (AccessDenied) when calling the CreateInstanceProfile operation | CreateInstanceProfile | arn:aws:iam::268367799918:role/* | keos |
| iam:AddRoleToInstanceProfile | not authorized to perform: iam:AddRoleToInstanceProfile on resource: instance profile eks-iam-for-sa-external-dns-role-eks-cl01 | AddRoleToInstanceProfile | arn:aws:iam::268367799918:instance-profile/* | keos |
| iam:ListRoleTags | not authorized to perform: iam:ListRoleTags on resource: role eks-iam-for-sa-external-dns-role-eks-cl01 | ListRoleTags | arn:aws:iam::268367799918:role/* | keos |
| route53:ListHostedZones | not authorized to perform: route53:ListHostedZones | ListHostedZones | * | keos |
| iam:PutRolePolicy | not authorized to perform: iam:PutRolePolicy on resource: role eks-iam-for-sa-external-dns-role-eks-cl01 | PutRolePolicy | arn:aws:iam::268367799918:role/* | keos |
| iam:UpdateAssumeRolePolicy | not authorized to perform: iam:UpdateAssumeRolePolicy on resource: role eks-iam-for-sa-external-dns-role-eks-cl01 | UpdateAssumeRolePolicy | arn:aws:iam::268367799918:role/* | keos |

**cluster-api-provider-aws v2.1.4-0.4.0**
| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| ec2:DescribeInstanceTypes | not authorized to perform: ec2:DescribeInstanceTypes | DescribeInstanceTypes | * | cluster-api-provider-aws |

**AWS Unmanaged**: Create EC2 cluster
| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| ec2:AuthorizeSecurityGroupIngress | failed to authorize security group "sg-04980f574d329b786" ingress rules | arn:aws:ec2:*:268367799918:security-group/* | cloud-provisioner |
| elasticloadbalancing:DescribeLoadBalancers | not authorized to perform: elasticloadbalancing:DescribeLoadBalancers | * | cloud-provisioner |
| elasticloadbalancing:CreateLoadBalancer | not authorized to perform: elasticloadbalancing:CreateLoadBalancer | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner |
| elasticloadbalancing:AddTags | Dependencie from CreateLoadBalancer | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner | 
| elasticloadbalancing:DescribeLoadBalancerAttributes | failed to describe classic load balancer "eks-cl02-apiserver" attributes | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner |
| elasticloadbalancing:DescribeTags | not authorized to perform: elasticloadbalancing:DescribeTags | * | cloud-provisioner |
| elasticloadbalancing:ModifyLoadBalancerAttributes | not authorized to perform: elasticloadbalancing:ModifyLoadBalancerAttributes | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner |
| elasticloadbalancing:RegisterInstancesWithLoadBalancer | not authorized to perform: elasticloadbalancing:RegisterInstancesWithLoadBalancer | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner |
| elasticloadbalancing:DeleteLoadBalancer | not authorized to perform: elasticloadbalancing:RegisterInstancesWithLoadBalancer | arn:aws:elasticloadbalancing:*:268367799918:loadbalancer/* | cloud-provisioner |
| ec2:RevokeSecurityGroupIngress | not authorized to perform this operation | arn:aws:ec2:*:268367799918:security-group/* | cloud-provisioner |

**Test**: keoscluster

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| eks:UpdateClusterConfig | Upgrade loggin | UpdateClusterConfig | arn:aws:eks:*:268367799918:cluster/* | cloud-provisioner |