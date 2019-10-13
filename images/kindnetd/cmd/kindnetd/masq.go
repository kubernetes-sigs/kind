/*
Copyright 2019 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"time"

	"github.com/coreos/go-iptables/iptables"
)

// NewIPMasqAgent returns a new IPMasqAgent
func NewIPMasqAgent(ipv6 bool, noMasqueradeCIDRs []string) *IPMasqAgent {
	protocol := iptables.ProtocolIPv4
	if ipv6 {
		protocol = iptables.ProtocolIPv6
	}
	ipt, _ := iptables.NewWithProtocol(protocol)

	// TODO: validate cidrs
	return &IPMasqAgent{
		iptables:          ipt,
		masqChain:         masqChainName,
		noMasqueradeCIDRs: noMasqueradeCIDRs,
	}
}

// IPMasqAgent is based on https://github.com/kubernetes-incubator/ip-masq-agent
// but collapsed into kindnetd and made ipv6 aware in an opionated and simplified
// fashion using "github.com/coreos/go-iptables" instead of k8s/pkg/util/iptables
type IPMasqAgent struct {
	iptables          *iptables.IPTables
	masqChain         string
	noMasqueradeCIDRs []string
}

// SyncRulesForever syncs ip masquerade rules forever
func (ma *IPMasqAgent) SyncRulesForever(interval time.Duration) {
	for {
		if err := ma.SyncRules(); err != nil {
			fmt.Printf("Error syncing iptables non-masq rules %v, retrying ... \n", err)
		}
		time.Sleep(interval)
	}
}

// name of nat chain for iptables masquerade rules
const masqChainName = "KIND-MASQ-AGENT"

// SyncRules syncs ip masquerade rules
func (ma *IPMasqAgent) SyncRules() error {
	// make sure our custom chain for non-masquerade exists
	exists := false
	chains, err := ma.iptables.ListChains("nat")
	if err != nil {
		return fmt.Errorf("failed to list chains: %v", err)
	}
	for _, ch := range chains {
		if ch == ma.masqChain {
			exists = true
			break
		}
	}
	if !exists {
		if err = ma.iptables.NewChain("nat", ma.masqChain); err != nil {
			return err
		}
	}

	// Packets to this network should not be masquerade, pods should be able to talk to other pods
	for _, cidr := range ma.noMasqueradeCIDRs {
		if err := ma.iptables.AppendUnique("nat", ma.masqChain, "-d", cidr, "-j", "RETURN", "-m", "comment", "--comment", "kind-masq-agent: local traffic is not subject to MASQUERADE"); err != nil {
			return err
		}
	}

	// Masquerade all the other traffic
	if err := ma.iptables.AppendUnique("nat", ma.masqChain, "-j", "MASQUERADE", "-m", "comment", "--comment", "kind-masq-agent: outbound traffic is subject to MASQUERADE (must be last in chain)"); err != nil {
		return err
	}

	// Send all non-LOCAL destination traffic to our custom KIND-MASQ-AGENT chain
	return ma.iptables.AppendUnique("nat", "POSTROUTING", "-m", "addrtype", "!", "--dst-type", "LOCAL", "-j", ma.masqChain, "-m", "comment", "--comment", "kind-masq-agent: ensure nat POSTROUTING directs all non-LOCAL destination traffic to our custom KIND-MASQ-AGENT chain")
}
