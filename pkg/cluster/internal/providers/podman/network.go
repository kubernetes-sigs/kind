/*
Copyright 2020 The Kubernetes Authors.

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

package podman

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"net"
	"regexp"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// This may be overridden by KIND_EXPERIMENTAL_PODMAN_NETWORK env,
// experimentally...
//
// By default currently picking a single network is equivalent to the previous
// behavior *except* that we moved from the default bridge to a user defined
// network because the default bridge is actually special versus any other
// docker network and lacks the embedded DNS
//
// For now this also makes it easier for apps to join the same network, and
// leaves users with complex networking desires to create and manage their own
// networks.
const fixedNetworkName = "kind"

// ensureNetwork creates a new network
// podman only creates IPv6 networks for versions >= 2.2.0
func ensureNetwork(name string) error {
	// network already exists
	if checkIfNetworkExists(name) {
		return nil
	}

	// in case kind is running in linux and IPv6 is disabled in linux kernel, then create the podman network without Ipv6
	// This is required here because i came across that even though ipv6 is disabled in kernel, podman doesn't stop you from creating a network with Ipv6 enabled,
	// which later will result in crashing when it'll try to create the cluster.
	if isIPv6DisabledInLinuxKernel() {
		return createNetwork(name, "")
	}

	// generate unique subnet per network based on the name
	// obtained from the ULA fc00::/8 range
	// Make N attempts with "probing" in case we happen to collide
	subnet := generateULASubnetFromName(name, 0)
	err := createNetwork(name, subnet)
	if err == nil {
		// Success!
		return nil
	}

	if isUnknownIPv6FlagError(err) ||
		isIPv6DisabledError(err) {
		return createNetwork(name, "")
	}

	// Only continue if the error is because of the subnet range
	// is already allocated
	if !isPoolOverlapError(err) {
		return err
	}

	// keep trying for ipv6 subnets
	const maxAttempts = 5
	for attempt := int32(1); attempt < maxAttempts; attempt++ {
		subnet := generateULASubnetFromName(name, attempt)
		err = createNetwork(name, subnet)
		if err == nil {
			// success!
			return nil
		} else if !isPoolOverlapError(err) {
			// unknown error ...
			return err
		}
	}
	return errors.New("exhausted attempts trying to find a non-overlapping subnet")

}

func createNetwork(name, ipv6Subnet string) error {
	if ipv6Subnet == "" {
		return exec.Command("podman", "network", "create", "-d=bridge", name).Run()
	}
	return exec.Command("podman", "network", "create", "-d=bridge",
		"--ipv6", "--subnet", ipv6Subnet, name).Run()
}

func checkIfNetworkExists(name string) bool {
	_, err := exec.Output(exec.Command(
		"podman", "network", "inspect",
		regexp.QuoteMeta(name),
	))
	return err == nil
}

func isUnknownIPv6FlagError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil &&
		strings.Contains(string(rerr.Output), "unknown flag: --ipv6")
}

func isIPv6DisabledError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil &&
		strings.Contains(string(rerr.Output), "is ipv6 enabled in the kernel")
}

func isIPv6DisabledInLinuxKernel() bool {

	// First check that we're running in Linux os

	output, err := exec.Output(exec.Command("podman", "info", "-f", "json"))
	if err != nil {
		return false
	}
	var result bool
	result = false
	var podmanInfo map[string]json.RawMessage
	var podmanVersion map[string]string
	json.Unmarshal(output, &podmanInfo)
	json.Unmarshal(podmanInfo["version"], &podmanVersion)
	// In case of linux, check if the ipv6 module is enabled in kernel
	if podmanVersion["Os"] == "linux" {
		// https://www.golinuxcloud.com/linux-check-ipv6-enabled/
		// one of 6 methods to check if ipv6 enabled in kernel, all are equivalent, but chose the following one  because it will work on all linux distribution, as the "cat" command utility is available on all linux distributions.
		decision, error := exec.Output(exec.Command("cat", "/sys/module/ipv6/parameters/disable"))
		// if the value is 0, then ipv6 module is enabled in linux kernel, otherwise, if it's 1 , it's disabled in kernel.
		if error == nil {
			if strings.Trim(string(decision), "\n") == "1" {
				result = true
			}
		}
	}
	return result
}

func isPoolOverlapError(err error) bool {
	rerr := exec.RunErrorForError(err)
	if rerr == nil {
		return false
	}
	output := string(rerr.Output)
	return strings.Contains(output, "is already used on the host or by another config") ||
		strings.Contains(output, "is being used by a network interface") ||
		strings.Contains(output, "is already being used by a cni configuration")
}

// generateULASubnetFromName generate an IPv6 subnet based on the
// name and Nth probing attempt
func generateULASubnetFromName(name string, attempt int32) string {
	ip := make([]byte, 16)
	ip[0] = 0xfc
	ip[1] = 0x00
	h := sha1.New()
	_, _ = h.Write([]byte(name))
	_ = binary.Write(h, binary.LittleEndian, attempt)
	bs := h.Sum(nil)
	for i := 2; i < 8; i++ {
		ip[i] = bs[i]
	}
	subnet := &net.IPNet{
		IP:   net.IP(ip),
		Mask: net.CIDRMask(64, 128),
	}
	return subnet.String()
}
