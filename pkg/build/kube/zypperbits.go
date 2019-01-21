/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kube

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// the packages repo
	repo = "https://download.opensuse.org/tumbleweed/repo/oss/"

	// packages necessary to install
	packages = "kubernetes-kubelet kubernetes-kubeadm kubernetes-client"
)

// ZypperBits implements Bits for the official upstream openSUSE packages
type ZypperBits struct {
}

var _ Bits = &ZypperBits{}

func init() {
	RegisterNamedBits("zypper", NewZypperBits)
}

// NewZypperBits returns a new Bits backed by the upstream debian packages
func NewZypperBits(kubeRoot string) (bits Bits, err error) {
	return &ZypperBits{}, nil
}

// Build implements Bits.Build
// for ZypperBits this does nothing
func (b *ZypperBits) Build() error {
	return nil
}

// Paths implements Bits.Paths
func (b *ZypperBits) Paths() map[string]string {
	return map[string]string{}
}

// Install implements Bits.Install
func (b *ZypperBits) Install(install InstallContext) error {
	addRepo := fmt.Sprintf("zypper --non-interactive --gpg-auto-import-keys addrepo %s repo-opensuse", repo)
	if err := install.Run("/bin/sh", "-c", addRepo); err != nil {
		log.Errorf("Adding Kubernetes apt repository failed! %v", err)
		return err
	}
	// install packages
	if err := install.Run("/bin/sh", "-c", fmt.Sprintf("zypper in -y %s", packages)); err != nil {
		log.Errorf("Installing Kubernetes packages failed! %v", err)
		return err
	}
	// get version for version file
	lines, err := install.CombinedOutputLines("/bin/sh", "-c", `kubelet --version`)
	if err != nil {
		log.Errorf("Failed to get Kubernetes version! %v", err)
		return err
	}
	// the output should be one line of the form `Kubernetes ${VERSION}`
	if len(lines) != 1 {
		log.Errorf("Failed to parse Kubernetes version with unexpected output: %v", lines)
		return fmt.Errorf("failed to parse Kubernetes version")
	}
	// write version file
	version := strings.SplitN(lines[0], " ", 2)[1]
	if err := install.Run("/bin/sh", "-c", fmt.Sprintf(`echo "%s" >> /kind/version`, version)); err != nil {
		log.Errorf("Failed to get Kubernetes version! %v", err)
		return err
	}
	return nil
}
