# Kindnet CNI plugin

Kubernetes container runtimes relies on the Container Network Interface (CNI) standard to manage Pod networking. While CNI provides a general framework for container networking, its full feature set isn't necessary for Kubernetes environments. This presents an opportunity for optimization.

At its core, Kubernetes has two primary networking requirements for CNI:

- IP Address Management (IPAM): Assign IP addresses to Pods.
- Interface Configuration: Create and configure network interfaces within Pods.

Many CNI plugins, however, are designed for more complex scenarios beyond Kubernetes' needs. This can lead to unnecessary overhead and complexity.

**NOTE** Usually the "CNI plugin" term is used to refer to "Kubernetes network plugins", that offer more networking capabilities, like Services, Network Policies, ... `kindnet` is a Network Plugin that uses cni-kindnet to assign IP to Pods and create its interfaces.

Traditional CNI plugins often rely on in-memory data structures or external daemon processes to manage network state. This can introduce challenges:

- Process Dependencies: Relying on daemons creates extra dependencies. If a daemon crashes or fails to restart correctly, it can disrupt network operations and complicate recovery.

- Reconciliation Overhead: Maintaining consistency between in-memory state and the actual network configuration requires complex reconciliation loops, which can consume resources and introduce delays.

`cni-kindnet` leverages SQLite3, a lightweight database, to maintain state and streamline operations. This eliminates the need for external daemons or complex chaining, resulting in a more efficient and reliable plugin.