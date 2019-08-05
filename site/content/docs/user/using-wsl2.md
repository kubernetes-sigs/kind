---
title: "Using WSL2"
menu:
  main:
    parent: "user"
    identifier: "using-wsl2"
    weight: 3
---
# Using WSL2

Kind can run using Windows Subsystem for Linux 2 (WSL2) on Windows 10 Insider builds. All the tools needed to build or run kind work in WSL2, but some extra steps are needed to switch to WSL2. This page covers these steps in brief but also links to the official documentation if you would like more details.

## Known Issues

- The default kernel shipped with WSL2 is missing a few options needed for Kubernetes to work correctly. Instructions for building an updated kernel are included below.


## Getting Windows 10 Insider Preview

Download latest ISO at https://www.microsoft.com/en-us/software-download/windowsinsiderpreviewadvanced . Choose "Windows 10 Insider Preview (FAST) - Build 18932". If there's a later build, that will work too.

### Installing on a virtual machine

> Note: this currently only works with Intel processors. The Hyper-V hypervisor used by WSL2 cannot run underneath another hypervisor on AMD processors.

Required Settings

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

Once your Windows Insider machine is ready, you need to do a few more steps to set up WSL2

1. Open a PowerShell window as an admin, then run
    ```PowerShell
    Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform, Microsoft-Windows-Subsystem-Linux
    ```
1. Reboot when prompted. 
1. After the reboot, set WSL to default to WSL2. Open an admin PowerShell window and run `wsl --set-default-version 2`.
1. Now, you can install your Linux distro of choice by searching the Windows Store. If you don't want to use the Windows Store, then follow the steps in the WSL docs for [manual install](https://docs.microsoft.com/en-us/windows/wsl/install-manual).
1. Start up your distro with the shortcut added to the start menu

### Updating Kernel

In Windows 10 Insider build 18941 and earlier, the WSL2 kernel is still missing a few features needed for kind to work correctly. A custom kernel is needed. Since WSL2 is installed and working, it's easy to build a new one with the right features included.

For the latest status on this, see [issue #707](https://github.com/kubernetes-sigs/kind/issues/707) and [microsoft/wsl#4165](https://github.com/microsoft/WSL/issues/4165). 

First, clone the latest WSL2-Linux-Kernel source and build it.

```bash
# This assumes Ubuntu or Debian, a different step may be needed for RPM based distributions
sudo apt install build-essential flex bison libssl-dev libelf-dev
git clone --depth 1 https://github.com/microsoft/WSL2-Linux-Kernel.git
cd WSL2-Linux-Kernel
make -j4 KCONFIG_CONFIG=Microsoft/config-wsl
mkdir /mnt/c/linuxtemp
cp arch/x86/boot/bzImage /mnt/c/linuxtemp/
```

Now, open an administrator PowerShell window and run these steps to apply the kernel:

```powershell
wsl.exe --shutdown
cd C:\WINDOWS\system32\lxss\tools
$acl = Get-Acl .\kernel
$acl.AddAccessRule( ( New-Object System.Security.AccessControl.FileSystemAccessRule(".\Administrators","FullControl","Allow") ) )
$acl | Set-Acl .\kernel
Move-Item kernel kernel.orig
Copy-Item c:\linuxtemp\bzImage kernel
```

Now, start a new WSL2 prompt and continue on with the steps to set up Docker.

## Setting up Docker in WSL2
1. Install Docker - here's links for [Debian](https://docs.docker.com/install/linux/docker-ce/debian/), [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/), and [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
1. Start the Docker daemon using init (not systemd) `sudo service docker start`. This needs to be done each time you start WSL2.

Now, move on to the [Quick Start](/docs/user/quick-start) to set up your cluster with kind.

## Helpful Tips for WSL2

- If you want to shutdown the WSL2 instance to save memory or "reboot", open an admin PowerShell prompt and run `wsl <distro> --shutdown`. Closing a WSL2 window doesn't shut it down automatically.
- You can check the status of all installed distros with `wsl --list --verbose`.
- If you had a distro installed with WSL1, you can convert it to WSL2 with `wsl --set-version <distro> 2`
