# AZURE Permissions

Requirements:
- Service Principal (cloud-provisioner creedentials)
  - Application Id = descriptor (client_id)
  - AAD - Application Secret ID = descriptor (client_secret)
- Resource group (capz for example) (control-plane and workers identities)
  - Managed Identity (capz-agentpool-restricted) --> Azure roles Assignments (capz-role-node)
  - Managed Identity (capz-control-plane-restricted) --> Azure roles Assignments (capz-role-control-plane)

### Permissions Table

**Test:** cloud-provisioner create cluster --name azure1 --vault-password 123456 --avoid-creation

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |

**Test:** cloud-provisioner create cluster --name azure1 --retain --vault-password 123456 --keep-mgmt

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| Microsoft.Authorization/roleAssignments/read | Read Secrets | Invalid client secret provided RESPONSE 403 Forbidden | Microsoft.Authorization | Provisioner |
| Microsoft.Resources/subscriptions/resourcegroups/read | Read ResourceGroup |does not have authorization to perform action 'Microsoft.Resources/subscriptions/resourcegroups/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourcegroups/azure1 | Microsoft.Resources | Provisioner |
| Microsoft.Resources/subscriptions/resourcegroups/write | Create Resourcegroup | does not have authorization to perform action 'Microsoft.Resources/subscriptions/resourcegroups/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourcegroups/xxxxxx | Microsoft.Resources | Provisioner |
| Microsoft.Network/virtualNetworks/read | Read virtualNetworks | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet | Microsoft.Network | Provisioner |
| Microsoft.Network/virtualNetworks/write | Create virtualNetworks | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet | Microsoft.Network | Provisioner |
| Microsoft.Network/networkSecurityGroups/read | Read networkSecurityGroups | does not have authorization to perform action 'Microsoft.Network/networkSecurityGroups/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkSecurityGroups/xxxxxx-node-nsg | Microsoft.Network | Provisioner |
| Microsoft.Network/networkSecurityGroups/write | Create networkSecurityGroups | does not have authorization to perform action 'Microsoft.Network/networkSecurityGroups/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkSecurityGroups/xxxxxx-node-nsg | Microsoft.Network | Provisioner |
| Microsoft.Network/routeTables/read | Read routeTables | does not have authorization to perform action 'Microsoft.Network/routeTables/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/routeTables/xxxxxx-node-routetable | Microsoft.Network | Provisioner |
| Microsoft.Network/routeTables/write | Create routeTables | does not have authorization to perform action 'Microsoft.Network/routeTables/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/routeTables/xxxxxx-node-routetable | Microsoft.Network | Provisioner |
| Microsoft.Network/publicIPAddresses/read | Read publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/publicIPAddresses/pip-xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/publicIPAddresses/write | Create publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/publicIPAddresses/pip-xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/natGateways/read | Read natGateways | does not have authorization to perform action 'Microsoft.Network/natGateways/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/natGateways/xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/natGateways/write | Create natGateways | does not have authorization to perform action 'Microsoft.Network/natGateways/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/natGateways/xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/virtualNetworks/subnets/read | Read subnets | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet/subnets/node-subnet | Microsoft.Network | Provisioner |
| Microsoft.Network/virtualNetworks/subnets/write | Create subnets | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet/subnets/node-subnet | Microsoft.Network | Provisioner |
| Microsoft.Network/natGateways/join/action | Join natGateways | does not have authorization to perform action 'Microsoft.Network/natGateways/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/natGateways/xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/networkSecurityGroups/join/action | Join networkSecurityGroups | does not have authorization to perform action 'Microsoft.Network/networkSecurityGroups/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkSecurityGroups/xxxxxx-node-nsg | Microsoft.Network | Provisioner |
| Microsoft.Network/routeTables/join/action | Join routeTables | does not have authorization to perform action 'Microsoft.Network/routeTables/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/routeTables/xxxxxx-node-routetable | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/read | Read loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/write | Create loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/join/action | Join loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/inboundNatRules/read | Read inboundNatRules | does not have authorization to perform action 'Microsoft.Network/loadBalancers/inboundNatRules/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb/inboundNatRules/xxxxxx-control-plane-8xhk2 | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/inboundNatRules/write | Create inboundNatRules | does not have authorization to perform action 'Microsoft.Network/loadBalancers/inboundNatRules/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb/inboundNatRules/xxxxxx-control-plane-8xhk2 | Microsoft.Network | Provisioner |
| Microsoft.Network/networkInterfaces/read | Read networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkInterfaces/xxxxxx-control-plane-8xhk2-nic | Microsoft.Network | Provisioner |
| Microsoft.Network/networkInterfaces/write | Create networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkInterfaces/xxxxxx-control-plane-8xhk2-nic | Microsoft.Network | Provisioner |
| Microsoft.Network/virtualNetworks/subnets/join/action | Join subnets | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet/subnets/control-plane-subnet | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/backendAddressPools/join/action | Join backendAddressPools | does not have authorization to perform action 'Microsoft.Network/loadBalancers/backendAddressPools/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb/backendAddressPools/xxxxxx-public-lb-backendPool | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/inboundNatRules/join/action | Join inboundNatRules | does not have authorization to perform action 'Microsoft.Network/loadBalancers/inboundNatRules/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb/inboundNatRules/xxxxxx-control-plane-8xhk2 | Microsoft.Network | Provisioner |
| Microsoft.Compute/virtualMachines/read | Read virtualMachines | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-control-plane-8xhk2 | Microsoft.Compute | Provisioner |
| Microsoft.Compute/virtualMachines/write | Create virtualMachines | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-control-plane-8xhk2 | Microsoft.Compute | Provisioner |
| Microsoft.Network/networkInterfaces/join/action | Join networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkInterfaces/xxxxxx-control-plane-8xhk2-nic | Microsoft.Network | Provisioner |
| Microsoft.Compute/virtualMachines/extensions/read | Read virtualMachines/extensions | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/extensions/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-control-plane-8xhk2/extensions/CAPZ.Linux.Bootstrapping | Microsoft.Compute | Provisioner |
| Microsoft.Compute/virtualMachines/extensions/write | Create virtualMachines/extensions | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/extensions/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-control-plane-8xhk2/extensions/CAPZ.Linux.Bootstrapping | Microsoft.Compute | Provisioner |
| Microsoft.ManagedIdentity/userAssignedIdentities/read | Read userAssignedIdentities | does not have authorization to perform action 'Microsoft.ManagedIdentity/userAssignedIdentities/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/capz/providers/Microsoft.ManagedIdentity/userAssignedIdentities/capz-agentpool | Microsoft.ManagedIdentity | Provisioner |
| Microsoft.ManagedIdentity/userAssignedIdentities/assign/action | Assign userAssignedIdentities | does not have permission to perform action 'Microsoft.ManagedIdentity/userAssignedIdentities/assign/action' on the linked scope(s) '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourcegroups/capz/providers/Microsoft.ManagedIdentity/userAssignedIdentities/capz-agentpool' | Microsoft.ManagedIdentity | Provisioner |


**Test:** cloud-provisioner create cluster --name azure1 --retain --vault-password 123456 (same permissions as --keep-mgmt) (same as above)

**Test:** clusterctl move --kubeconfig remote_kubeconfig --to-kubeconfig local_kubeconfig --namespace cluster-azure1 --dry-run
Performing move...
********************************************************
This is a dry-run move, will not perform any real action
********************************************************
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ clusterctl move --kubeconfig remote_kubeconfig --to-kubeconfig local_kubeconfig --namespace cluster-azure1 (no needed additonal permissions)  
Performing move...  
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ clusterctl move --to-kubeconfig remote_kubeconfig --kubeconfig local_kubeconfig --namespace cluster-azure1
Performing move...  
Discovering Cluster API objects  
Moving Cluster API objects Clusters=1  
Moving Cluster API objects ClusterClasses=0  
Creating objects in the target cluster  
Deleting objects from the source cluster  

❯ cluclusterctl --kubeconfig remote_kubeconfig describe cluster azure1 -n cluster-azure1  
NAME                                                       READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/azure1                                             True                     118m
├─ClusterInfrastructure - AzureCluster/azure1              True                     118m
├─ControlPlane - KubeadmControlPlane/azure1-control-plane  True                     118m
│ └─3 Machines...                                          True                     118m
└─Workers
  ├─MachineDeployment/azure1w1-md-0                        True                     118m
  │ └─Machine/azure1w1-md-0-7fddd56cbdxcw8c4-x2648         True                     118m
  ├─MachineDeployment/azure1w1-md-1                        True                     13m
  │ └─5 Machines...                                        True                     24m    
  └─MachineDeployment/azure1w1-md-2                        True                     118m
    └─Machine/azure1w1-md-2-7c9f4b574x96488-fr6xx          True                     118m 

**Test:** Auto-Scale (up/down) (deploy "n" nginx pods and check if the nodes are scaled up)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 6 | 1 | 6 | 5 | ScalingUp | 3m53s | v1.24.10 | 
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | 0 | Running   | 3m50s | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 2 | 1 | 2 | 1 | ScalingUp | 3m51s | v1.24.10 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 6 | 6 | 6 | |0 | Running | 6m27s | v1.24.10 |
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | |0 | Running | 6m24s | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 2 | 2 | 2 | |0 | Running | 6m25s | v1.24.10 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 6 | 1 | 6 | 5 | ScalingDown | 19m | v1.24.10 | 
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | 0 | Running   | 19m | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 2 | 1 | 2 | 1 | ScalingDown | 19m | v1.24.10 |

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| Microsoft.Compute/virtualMachines/delete | Delete virtualMachines | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-1-md-0 | Microsoft.Compute | Provisioner |
| Microsoft.Compute/virtualMachines/extensions/delete | Delete virtualMachines/extensions | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/extensions/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-md-0/extensions/CAPZ.Linux.Bootstrapping | Microsoft.Compute | Provisioner |
| Microsoft.Compute/disks/delete | Delete disks | does not have authorization to perform action 'Microsoft.Compute/disks/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/disks/xxxxxx--md-0_OsDisk | Microsoft.Compute | Provisioner |


**Test:** Scale Manually (up/dpwn) kubectl -n cluster-azure1 scale --replicas 3 machinedeployments --all (same as above)

**Test:** Create new MachineDeployment Manually (same as above) (scale ok)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-3 | azure1 | 1 | 1 | 1 | |0 | Running | 48s | v1.24.10 |

**Test:** Destroy Machine on Azure UI (self-healing) (azure1w1-md-2) (same as above)

| NAMESPACE | NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE  | AGE | VERSION
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| cluster-azure1 | azure1w1-md-0 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-1 | azure1 | 1 | 1 | 1 | |0 | Running | 26m | v1.24.10 |
| cluster-azure1 | azure1w1-md-2 | azure1 | 1 | 1 | 1 | |0 |  ScalingUp| 64m | v1.24.10 |
| cluster-azure1 | azure1w1-md-3 | azure1 | 1 | 1 | 1 | |0 | Running | 48s | v1.24.10 |

**Test:** kubectl --kubeconfig local_kubeconfig -n cluster-azure1 patch KubeadmControlPlane azure1-control-plane --type merge -p '{"spec":{"version":"v1.25.9"}}' 
(upgrade control-plane k8s version from 1.24.10 to 1.25.9) (no more permissions needed)

**Test:** kubectl --kubeconfig local_kubeconfig -n cluster-azure1 patch machinedeployments.cluster.x-k8s.io azure1w1-md-0 --type merge -p '{"spec":{"template":{"spec":{"version":"v1.25.9"}}}}'  (no more permissions needed)  

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| azure1w1-md-0 |  azure1 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.10  |
| azure1w1-md-1 |  azure1 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.10  |
| azure1w1-md-2 |  azure1 | 1 | 1 | 1 | 0 | Running | 17m | v1.24.10  |

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| azure1w1-md-0 | azure1 | 2 | 2 | 1 | 0 | Running     | 18m | v1.25.9  |
| azure1w1-md-1 | azure1 | 1 | 1 | 1 | 0 | Running     | 18m | v1.24.10 |
| azure1w1-md-2 | azure1 | 1 | 1 | 1 | 0 | Running     | 18m | v1.24.10 |

| NAME | CLUSTER | REPLICAS | READY | UPDATED | UNAVAILABLE | PHASE | AGE | VERSION |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| azure1w1-md-0 | azure1 | 1 | 1 | 1 | 0 | Running | 21m | v1.25.9  |
| azure1w1-md-1 | azure1 | 1 | 1 | 1 | 0 | Running | 21m | v1.24.10 | 
| azure1w1-md-2 | azure1 | 1 | 1 | 1 | 0 | Running | 21m | v1.24.10 | 


**Test:** Delete cluster (From local container)
kubectl --kubeconfig local_kubeconfig -n cluster-azure1 delete cluster azure1

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| Microsoft.Resources/subscriptions/resourcegroups/delete | Delete ResourceGroup | does not have authorization to perform action 'Microsoft.Resources/subscriptions/resourcegroups/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx | Microsoft.Resources | Provisioner |
| Microsoft.Compute/disks/delete | Delete disks | does not have authorization to perform action 'Microsoft.Compute/disks/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/disks/xxxxxx-md-1-mdqww_OSDisk | Microsoft.Compute | Provisioner |
| Microsoft.Network/virtualNetworks/delete | Delete virtualNetworks | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet | Microsoft.Network | Provisioner |
| Microsoft.Network/networkSecurityGroups/delete | Delete networkSecurityGroups | does not have authorization to perform action 'Microsoft.Network/networkSecurityGroups/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkSecurityGroups/xxxxxx-node-nsg | Microsoft.Network | Provisioner |
| Microsoft.Network/routeTables/delete | Delete routeTables | does not have authorization to perform action 'Microsoft.Network/routeTables/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/routeTables/xxxxxx-node-routetable | Microsoft.Network | Provisioner |
| Microsoft.Network/publicIPAddresses/delete | Delete publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/publicIPAddresses/pip-xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/natGateways/delete | Delete natGateways | does not have authorization to perform action 'Microsoft.Network/natGateways/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/natGateways/xxxxxx-node-natgw-1 | Microsoft.Network | Provisioner |
| Microsoft.Network/virtualNetworks/subnets/delete | Delete subnets | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/virtualNetworks/xxxxxx-vnet/subnets/node-subnet | Microsoft.Network | Provisioner |
| Microsoft.Network/loadBalancers/delete | Delete loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/loadBalancers/xxxxxx-public-lb | Microsoft.Network | Provisioner |
| Microsoft.Network/networkInterfaces/delete | Delete networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/networkInterfaces/xxxxxx-control-plane-8xhk2-nic | Microsoft.Network | Provisioner |
| Microsoft.Compute/virtualMachines/delete | Delete virtualMachines | does not have authorization to perform action 'Microsoft.Compute/virtualMachines/delete' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/virtualMachines/xxxxxx-control-plane-8xhk2 | Microsoft.Compute | Provisioner |

**Test**: Keos Install

| Permission | Needed for | Description | Resource | Application |
| --- | --- | --- | --- | --- |
| kubelet Failed to pull image "eosregistry.azurecr.io/keos/stratio/capsule:0.1.1-0.3.1 | pull image "eosregistry.azurecr.io/keos/stratio/capsule:0.1.1-0.3.1 | Microsoft.ContainerRegistry/registries/pull/read | Microsoft.ContainerRegistry/registries/pull/read | keos (workers) |
| Microsoft.Network/publicIPAddresses/write | Create publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Network/publicIPAddresses/azure1-a3832d7641b0f422cadfe775c6e96cb9 | Microsoft.Network | keos (Control-plane) |
| Microsoft.Compute/disks/read | /subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Compute/disks/pvc-4816139c-df5c-43c9-bc51-394164b1522c | Microsoft.Compute | keos (Control-plane & workers) |
| Microsoft.Compute/disks/write | Create disks | does not have authorization to perform action 'Microsoft.Compute/disks/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/xxxxxx/providers/Microsoft.Compute/disks/pvc-4816139c-df5c-43c9-bc51-394164b1522c | Microsoft.Compute | keos (Control-plane & workers) |
| Microsoft.Network/dnsZones/read | Read dnsZones | does not have authorization to perform action 'Microsoft.Network/dnsZones/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/dnsZones/domain.ext' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/privateDnsZones/read | Read privateDnsZones | does not have authorization to perform action 'Microsoft.Network/privateDnsZones/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/privateDnsZones/domain.ext' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/publicIPAddresses/read | Read publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/publicIPAddresses/azure1-a3832d7641b0f422cadfe775c6e96cb9 | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/publicIPAddresses/write | Create publicIPAddresses | does not have authorization to perform action 'Microsoft.Network/publicIPAddresses/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/publicIPAddresses/azure1-a3832d7641b0f422cadfe775c6e96cb9 | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/networkSecurityGroups/read | Read networkSecurityGroups | does not have authorization to perform action 'Microsoft.Network/networkSecurityGroups/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/dnsZones/domain.ext' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/loadBalancers/write | Create loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/loadBalancers/azure1' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/loadBalancers/read | Read loadBalancers | does not have authorization to perform action 'Microsoft.Network/loadBalancers/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/loadBalancers/azure1' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/networkInterfaces/read | Read networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/read' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/networkInterfaces/azure1w1-md-2-c59n2-nic' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/networkInterfaces/write | Create networkInterfaces | does not have authorization to perform action 'Microsoft.Network/networkInterfaces/write' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/networkInterfaces/azure1w1-md-2-c59n2-nic' | Microsoft.Network | keos (Control-plane) |
| Microsoft.Network/virtualNetworks/subnets/join/action | Join subnets | does not have authorization to perform action 'Microsoft.Network/virtualNetworks/subnets/join/action' over scope '/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/azure1/providers/Microsoft.Network/virtualNetworks/azure1/subnets/node-subnet' | Microsoft.Network | keos (Control-plane) |
