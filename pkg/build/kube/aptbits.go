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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// AptBits implements Bits for the official upstream debian packages
type AptBits struct {
}

var _ Bits = &AptBits{}

func init() {
	RegisterNamedBits("apt", NewAptBits)
}

// NewAptBits returns a new Bits backed by the upstream debian packages
func NewAptBits(kubeRoot string) (bits Bits, err error) {
	return &AptBits{}, nil
}

// Build implements Bits.Build
// for AptBits this does nothing
func (b *AptBits) Build() error {
	return nil
}

// Paths implements Bits.Paths
func (b *AptBits) Paths() map[string]string {
	return map[string]string{}
}

// Install implements Bits.Install
func (b *AptBits) Install(install InstallContext) error {
	// add apt repo
	addKey := `curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -`
	addSources := `cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF`
	if err := install.Run("/bin/sh", "-c", addKey); err != nil {
		log.Errorf("Adding Kubernetes apt key failed! %v", err)
		return err
	}
	if err := install.Run("/bin/sh", "-c", addSources); err != nil {
		log.Errorf("Adding Kubernetes apt repository failed! %v", err)
		return err
	}
	// install packages
	if err := install.Run("/bin/sh", "-c", `clean-install kubelet kubeadm kubectl`); err != nil {
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
		return errors.New("failed to parse Kubernetes version")
	}
	// write version file
	version := strings.SplitN(lines[0], " ", 2)[1]
	if err := install.Run("/bin/sh", "-c", fmt.Sprintf(`echo "%s" >> /kind/version`, version)); err != nil {
		log.Errorf("Failed to get Kubernetes version! %v", err)
		return err
	}
	return nil
}
