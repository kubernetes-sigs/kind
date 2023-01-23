---
title: "Using WSL2"
menu:
  main:
    parent: "user"
    identifier: "using-wsl2"
    weight: 3
description: |-
  Kind can run using Windows Subsystem for Linux 2 (WSL2) on Windows 10 May 2020 Update (build 19041). 
  
  All the tools needed to build or run kind work in WSL2, but some extra steps are needed to switch to WSL2. This page covers these steps in brief but also links to the official documentation if you would like more details.
---

## Getting Windows 10

Download latest ISO at https://www.microsoft.com/en-us/software-download/windows10ISO. Choose "Windows 10 May 2020 Update". If there's a later update, that will work too.

### Installing on a virtual machine

Required Settings

- Supported processor and operating system, see [Enable Nested Virtualization](https://learn.microsoft.com/en-us/virtualization/hyper-v-on-windows/user-guide/nested-virtualization) guide for Windows
  - Intel processors require Windows 10/Windows Server 2016 or greater and the processor must support VT-x and extended page tables (also known as [second level address translation](https://en.wikipedia.org/wiki/Second_Level_Address_Translation))
  - AMD processors require Windows 11/Windows Server 2022 or greater and the processor generation must be AMD EPYC or Ryzen or newer
- At least 8GB of memory
  - It's best to use a static memory allocation, not dynamic. The VM will automatically use paging inside so you don't want it to page on the VM host.
- Enable nested virtualization support. On Hyper-V, you need to run this from an admin PowerShell prompt - `Set-VMProcessor -VMName ... -ExposeVirtualizationExtensions $true`
- Attach the ISO to a virtual DVD drive
- Create a virtual disk with at least 80GB of space

Now, start up the VM. Watch carefully for the "Press any key to continue installation..." screen so you don't miss it. Windows Setup will start automatically.

### Installing on a physical machine

If you're using a physical machine, you can mount the ISO, copy the files to a FAT32 formatted USB disk, and boot from that instead. Be sure the machine is configured to boot using UEFI (not legacy BIOS), and has Intel VT or AMD-V enabled for the hypervisor.

### Tips during setup

- You can skip the product key page
- On the "Sign in with Microsoft" screen, look for the "offline account" button.

## Setting up WSL2

If you want the full details, see the [Installation Instructions for WSL2](https://docs.microsoft.com/en-us/windows/wsl/wsl2-install). This is the TL;DR version.

Once your Windows machine is ready, you need to do a few more steps to set up WSL2

1. Open a PowerShell window as an admin, then run

    {{< codeFromInline lang="powershell" >}}
Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform, Microsoft-Windows-Subsystem-Linux
{{< /codeFromInline >}}

1. Reboot when prompted.
1. After the reboot, set WSL to default to WSL2. Open an admin PowerShell window and run
    {{< codeFromInline lang="powershell" >}}
wsl --set-default-version 2
{{< /codeFromInline >}}
1. Now, you can install your Linux distro of choice by searching the Windows Store. If you don't want to use the Windows Store, then follow the steps in the WSL docs for [manual install](https://docs.microsoft.com/en-us/windows/wsl/install-manual).
1. Start up your distro with the shortcut added to the start menu

## Setting up Docker in WSL2 with Docker Desktop

Install Docker with WSL2 backend here: https://docs.docker.com/docker-for-windows/wsl/


## Setting up Docker in WSL2 without Docker Desktop

Alternatively, docker can be installed in WSL2 without using Docker Desktop.
See for example: https://dev.to/bowmanjd/install-docker-on-windows-wsl-without-docker-desktop-34m9


Now, move on to the [Quick Start](/docs/user/quick-start) to set up your cluster with kind.

## Accessing a Kubernetes Service running in WSL2

1. prepare cluster config with exported node port
    {{< codeFromInline lang="yaml" >}}
# cluster-config.yml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30000
    hostPort: 30000
    protocol: TCP
{{< /codeFromInline >}}

1. create cluster `kind create cluster --config=cluster-config.yml`
1. create deployment `kubectl create deployment nginx --image=nginx --port=80`
1. create service `kubectl create service nodeport nginx --tcp=80:80 --node-port=30000`
1. access service `curl localhost:30000`

Alternatively, see [Helpful Tips for WSL2](#helpful-tips-for-wsl2)

## Kubernetes Service with Session Affinity

If you want to create a Kubernetes Service with `sessionAffinity: ClientIP` it will not be accessible (and neither will any Service created afterwards).
WSL2 kernel is missing `xt_recent` kernel module, which is used by Kube Proxy to implement session affinity. You need to compile a custom kernel to enable this feature.

1. Build a kernel with `xt_recent` kernel module enabled
    {{< codeFromInline lang="bash" >}}
docker run --name wsl-kernel-builder --rm -it ubuntu:latest bash

WSL_COMMIT_REF=linux-msft-5.4.72 # change this line to the version you want to build

# Install dependencies
apt update
apt install -y git build-essential flex bison libssl-dev libelf-dev bc

# Checkout WSL2 Kernel repo
mkdir src
cd src
git init
git remote add origin https://github.com/microsoft/WSL2-Linux-Kernel.git
git config --local gc.auto 0
git -c protocol.version=2 fetch --no-tags --prune --progress --no-recurse-submodules --depth=1 origin +${WSL_COMMIT_REF}:refs/remotes/origin/build/linux-msft-wsl-5.4.y
git checkout --progress --force -B build/linux-msft-wsl-5.4.y refs/remotes/origin/build/linux-msft-wsl-5.4.y

# Enable xt_recent kernel module
sed -i 's/# CONFIG_NETFILTER_XT_MATCH_RECENT is not set/CONFIG_NETFILTER_XT_MATCH_RECENT=y/' Microsoft/config-wsl

# Compile the kernel 
make -j2 KCONFIG_CONFIG=Microsoft/config-wsl

# From the host terminal copy the newly built kernel
docker cp wsl-kernel-builder:/src/arch/x86/boot/bzImage .
{{< /codeFromInline >}}
1. Configure WSL to use newly built kernel: https://docs.microsoft.com/en-us/windows/wsl/wsl-config#configure-global-options-with-wslconfig

   Create a `.wslconfig` file in `C:\Users\<your-user-name>\`:
    {{< codeFromInline lang="toml" >}}
[wsl2]
kernel=c:\\path\\to\\your\\kernel\\bzImage
{{< /codeFromInline >}}

## Helpful Tips for WSL2

- If you want to terminate the WSL2 instance to save memory or "reboot", open an admin PowerShell prompt and run `wsl --terminate <distro>`. Closing a WSL2 window doesn't shut it down automatically.
- You can check the status of all installed distros with `wsl --list --verbose`.
- If you had a distro installed with WSL1, you can convert it to WSL2 with `wsl --set-version <distro> 2`
- Alternative of [Accessing a Kubernetes Service running in WSL2](#accessing-a-kubernetes-service-running-in-wsl2) or [Setting Up An Ingress Controller](/docs/user/ingress/#setting-up-an-ingress-controller) for accessing workloads is using `kubectl port-forward --address=0.0.0.0`.
