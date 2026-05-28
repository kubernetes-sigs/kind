/*
Copyright 2026 The Kubernetes Authors.

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

// Package swarm is the multi-host kind provider built on top of an
// existing Docker Swarm.  Containers for each node live on different
// hosts and share one overlay network ("kind") so they reach each
// other on overlay IPs (10.0.x.x) regardless of physical placement.
//
// User responsibilities (the provider does NOT bootstrap the swarm):
//   - Each host must run a Docker daemon.
//   - A Docker Swarm must already be initialised across them.
//   - For every host containing nodes, a `docker context` named after
//     that host must exist on the machine running kind.  Example:
//       docker context create worker-1 --docker host=ssh://root@worker-1...
//   - A single overlay network named "kind" must exist on the swarm.
//
// kind selects this provider via KIND_EXPERIMENTAL_PROVIDER=swarm or
// via the `--multihost` flag.  Hosts are supplied via the
// KIND_HOSTS environment variable (or `--hosts` CLI flag).
//
// Format of host list:  <ctx>=<addr>[,<ctx>=<addr>...]
//
//   - <ctx> is a docker context name (use "default" for the local daemon).
//   - <addr> is the host's externally-reachable IP (used for kubeconfig).
//
// First entry in the list is treated as the swarm manager (and as
// hosts[0] for the round-robin distribution of node containers).
package swarm

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// Host identifies one machine in the swarm: a docker context name
// to operate on its daemon plus an externally-reachable address.
type Host struct {
	Context string // docker context name; "default" = local daemon
	Addr    string // host IP/hostname reachable from outside the swarm
}

// HostsFromEnv reads the host list from KIND_HOSTS (preferred) or
// from the supplied raw string.  Returns nil if neither is set.
func HostsFromEnv(raw string) ([]Host, error) {
	if raw == "" {
		raw = os.Getenv("KIND_HOSTS")
	}
	if raw == "" {
		return nil, nil
	}
	return ParseHosts(raw)
}

// ParseHosts turns "ctx1=addr1,ctx2=addr2" into []Host.  First entry
// is treated as the swarm manager.
func ParseHosts(s string) ([]Host, error) {
	if s == "" {
		return nil, fmt.Errorf("empty host list")
	}
	var out []Host
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 || eq == len(pair)-1 {
			return nil, fmt.Errorf("invalid host spec %q (want <ctx>=<addr>)", pair)
		}
		out = append(out, Host{
			Context: strings.TrimSpace(pair[:eq]),
			Addr:    strings.TrimSpace(pair[eq+1:]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid hosts parsed from %q", s)
	}
	return out, nil
}

// dockerArgs prefixes a docker command with --context=<name>.
// Centralising it makes the call sites readable and keeps the
// difference from the local docker provider one flag wide.
func dockerArgs(ctxName string, args ...string) []string {
	return append([]string{"--context", ctxName}, args...)
}

// mergeHosts builds a combined host list from a YAML-derived list and the
// CLI-derived list (--hosts), preserving the YAML order.  When the same
// context appears in both, the CLI entry's Addr wins (it's usually the
// reachable IP set by the operator), but the YAML entry's position drives
// the manager (hosts[0]) choice.
func mergeHosts(fromYAML []config.Host, fromCLI []Host) []Host {
	out := make([]Host, 0, len(fromYAML))
	seen := make(map[string]int, len(fromYAML))
	cliByCtx := make(map[string]Host, len(fromCLI))
	for _, h := range fromCLI {
		cliByCtx[h.Context] = h
	}
	for _, y := range fromYAML {
		h := Host{Context: y.Context, Addr: y.Addr}
		if c, ok := cliByCtx[y.Context]; ok && c.Addr != "" {
			h.Addr = c.Addr
		}
		seen[h.Context] = len(out)
		out = append(out, h)
	}
	for _, c := range fromCLI {
		if _, ok := seen[c.Context]; ok {
			continue
		}
		out = append(out, c)
	}
	return out
}
