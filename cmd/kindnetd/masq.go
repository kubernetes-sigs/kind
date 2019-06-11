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
	"bytes"
	"fmt"
	"strings"
	"time"

	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"
)

// NewIPMasqAgent returns a new IPMasqAgent
func NewIPMasqAgent(ipv6 bool, noMasqueradeCIDRs []string) (*IPMasqAgent, error) {
	execer := utilexec.New()
	dbus := utildbus.New()
	protocol := utiliptables.ProtocolIpv4
	if ipv6 {
		protocol = utiliptables.ProtocolIpv6
	}
	masqChain := utiliptables.Chain(masqChainName)
	iptables := utiliptables.New(execer, dbus, protocol)

	// TODO: validate cidrs
	return &IPMasqAgent{
		iptables:          iptables,
		masqChain:         masqChain,
		noMasqueradeCIDRs: noMasqueradeCIDRs,
	}, nil
}

// IPMasqAgent is based on https://github.com/kubernetes-incubator/ip-masq-agent
// but collapsed into kindnetd and made ipv6 aware in an opionated and simplified
// fashion
type IPMasqAgent struct {
	iptables          utiliptables.Interface
	masqChain         utiliptables.Chain
	noMasqueradeCIDRs []string
}

// SyncRulesForever syncs ip masquerade rules forever
func (ma *IPMasqAgent) SyncRulesForever(interval time.Duration) error {
	for {
		if err := ma.SyncRules(); err != nil {
			return err
		}
		time.Sleep(interval)
	}
}

// name of nat chain for iptables masquerade rules
const masqChainName = "KIND-MASQ-AGENT"

// SyncRules syncs ip masquerade rules
func (ma *IPMasqAgent) SyncRules() error {
	// TODO(aojea): don´t sync if there are no changes
	// make sure our custom chain for non-masquerade exists
	ma.iptables.EnsureChain(utiliptables.TableNAT, ma.masqChain)

	// ensure that any non-local in POSTROUTING jumps to masqChain
	if err := ensurePostroutingJump(ma.iptables, ma.masqChain); err != nil {
		return err
	}

	// build up lines to pass to iptables-restore
	lines := bytes.NewBuffer(nil)
	writeLine(lines, "*nat")
	writeLine(lines, utiliptables.MakeChainLine(ma.masqChain)) // effectively flushes masqChain atomically with rule restore

	// non-masquerade for user-provided CIDRs
	for _, cidr := range ma.noMasqueradeCIDRs {
		writeNonMasqRule(ma.masqChain, lines, cidr)
	}

	// masquerade all other traffic that is not bound for a --dst-type LOCAL destination
	writeMasqRule(ma.masqChain, lines)

	writeLine(lines, "COMMIT")
	if err := ma.iptables.RestoreAll(lines.Bytes(), utiliptables.NoFlushTables, utiliptables.NoRestoreCounters); err != nil {
		return err
	}
	return nil
}

// NOTE(mtaufen): iptables requires names to be <= 28 characters, and somehow prepending "-m comment --comment " to this string makes it think this condition is violated
// Feel free to dig around in iptables and see if you can figure out exactly why; I haven't had time to fully trace how it parses and handle subcommands.
// If you want to investigate, get the source via `git clone git://git.netfilter.org/iptables.git`, `git checkout v1.4.21` (the version I've seen this issue on,
// though it may also happen on others), and start with `git grep XT_EXTENSION_MAXNAMELEN`.
func postroutingJumpComment(masqChain utiliptables.Chain) string {
	return fmt.Sprintf("kind-masq-agent: ensure nat POSTROUTING directs all non-LOCAL destination traffic to our custom %s chain", masqChain)
}

func ensurePostroutingJump(iptables utiliptables.Interface, masqChain utiliptables.Chain) error {
	if _, err := iptables.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainPostrouting,
		"-m", "comment", "--comment", postroutingJumpComment(masqChain),
		"-m", "addrtype", "!", "--dst-type", "LOCAL", "-j", string(masqChain)); err != nil {
		return fmt.Errorf("failed to ensure that %s chain %s jumps to MASQUERADE: %v", utiliptables.TableNAT, masqChain, err)
	}
	return nil
}

const nonMasqRuleComment = `-m comment --comment "kind-masq-agent: local traffic is not subject to MASQUERADE"`

func writeNonMasqRule(masqChain utiliptables.Chain, lines *bytes.Buffer, cidr string) {
	writeRule(lines, utiliptables.Append, masqChain, nonMasqRuleComment, "-d", cidr, "-j", "RETURN")
}

const masqRuleComment = `-m comment --comment "ip-masq-agent: outbound traffic is subject to MASQUERADE (must be last in chain)"`

func writeMasqRule(masqChain utiliptables.Chain, lines *bytes.Buffer) {
	writeRule(lines, utiliptables.Append, masqChain, masqRuleComment, "-j", "MASQUERADE")
}

// Similar syntax to utiliptables.Interface.EnsureRule, except you don't pass a table
// (you must write these rules under the line with the table name)
func writeRule(lines *bytes.Buffer, position utiliptables.RulePosition, chain utiliptables.Chain, args ...string) {
	fullArgs := append([]string{string(position), string(chain)}, args...)
	writeLine(lines, fullArgs...)
}

// Join all words with spaces, terminate with newline and write to buf.
func writeLine(lines *bytes.Buffer, words ...string) {
	lines.WriteString(strings.Join(words, " ") + "\n")
	fmt.Printf("Handling iptables: %s\n", lines)
}
