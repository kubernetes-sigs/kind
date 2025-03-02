/*
Copyright 2025 The Kubernetes Authors.

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
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/netip"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
)

func getInterfaceName() string {
	rndString := make([]byte, 4)
	_, err := rand.Read(rndString)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("knet%x", rndString)
}

func writeNetworkConfigWithoutIPs(n *NetworkConfig) error {
	// Insert the subnet into the database if it doesn't exist
	_, err := db.Exec(`
		INSERT INTO pods (
			container_id, name, namespace, uid, netns,
			interface_name,	interface_mtu
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, n.ContainerID, n.Name, n.Namespace, n.UID, n.NetNS,
		n.InterfaceName, n.MTU)
	if err != nil {
		return fmt.Errorf("error inserting IP range: %w", err)
	}
	return nil
}

func writeNetworkConfigPortmaps(n *NetworkConfig) error {
	if len(n.PortMaps) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
				INSERT INTO portmap_entries (
					container_id, host_ip, host_port, protocol, container_ip, container_port
				) VALUES (?, ?, ?, ?, ?, ?)
			`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range n.PortMaps {
		_, err = stmt.Exec(
			n.ContainerID, entry.HostIP, entry.HostPort,
			entry.Protocol, entry.ContainerIP, entry.ContainerPort,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing statement: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func newNetworkConfig(args *skel.CmdArgs) (*NetworkConfig, error) {
	conf := KindnetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}

	// check if the cidrs are already stored in the db
	if err := createIPRanges(conf.Ranges); err != nil {
		return nil, fmt.Errorf("failed to configure network ranges: %v", err)
	}

	result := &NetworkConfig{
		ContainerID:   args.ContainerID,
		NetNS:         args.Netns,
		InterfaceName: getInterfaceName(),
		MTU:           getDefaultGwInterfaceMTU(),
	}

	k8sArgs := K8sArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		logger.Printf("could not load kubernetes metadata: %v", err)
	} else {
		result.Namespace = string(k8sArgs.K8S_POD_NAMESPACE)
		result.Name = string(k8sArgs.K8S_POD_NAME)
		result.UID = string(k8sArgs.K8S_POD_UID)
	}

	// Write the existing Pods before allocating the IPs
	err := writeNetworkConfigWithoutIPs(result)
	if err != nil {
		return nil, err
	}

	// obtain IPs for the pods and update the database
	err = getIPConfig(result)
	if err != nil {
		return nil, fmt.Errorf("could not get IP configuration: %v", err)
	}

	// process the portmap entries and store then wiht a format
	// that no need processing to install in the nftables rules.
	// Basically we need to infer the HostIP when is empty, it should
	// be 0.0.0.0/0 or ::/0 depending on the IP family to match all
	// addresses.
	// Also, discard possible mismatches between HostIP and ContainerIP.
	for _, portmap := range conf.RuntimeConfig.PortMaps {
		entry := PortMapConfig{
			HostPort:      portmap.HostPort,
			Protocol:      portmap.Protocol,
			ContainerPort: portmap.ContainerPort,
		}

		if portmap.HostIP != "" {
			ip, err := netip.ParseAddr(portmap.HostIP)
			if err != nil {
				continue
			}
			if ip.Is4() && result.IPv4 != nil {
				entry.HostIP = ip.String()
				entry.ContainerIP = result.IPv4.String()
			}
			if ip.Is6() && result.IPv6 != nil {
				entry.HostIP = ip.String()
				entry.ContainerIP = result.IPv6.String()
			}
		} else if result.IPv4 != nil && result.IPv6 != nil {
			// This is an special case as we need to store two entries
			entry2 := entry
			entry2.HostIP = "::"
			entry2.ContainerIP = result.IPv6.String()
			result.PortMaps = append(result.PortMaps, entry2)

			entry.HostIP = "0.0.0.0"
			entry.ContainerIP = result.IPv4.String()
		} else if result.IPv4 != nil {
			entry.HostIP = "0.0.0.0"
			entry.ContainerIP = result.IPv4.String()
		} else if result.IPv6 != nil {
			entry.HostIP = "::"
			entry.ContainerIP = result.IPv6.String()
		}
		result.PortMaps = append(result.PortMaps, entry)
	}

	// Write the Portmap entries after we got the Pod IPs
	err = writeNetworkConfigPortmaps(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// IPRange represents an IP range from the ipam_ranges table.
type IPRange struct {
	Subnet string
}

func createIPRanges(ipRanges []string) error {
	for _, ipRange := range ipRanges {
		// Trim any leading/trailing whitespace
		ipRange = strings.TrimSpace(ipRange)

		// Parse the IP range (assuming CIDR notation)
		_, _, err := net.ParseCIDR(ipRange)
		if err != nil {
			return fmt.Errorf("error parsing IP range %s: %w", ipRange, err)
		}

		// Insert the subnet into the database if it doesn't exist
		_, err = db.Exec(`
			INSERT INTO ipam_ranges (subnet)
			SELECT ?
			WHERE NOT EXISTS(SELECT 1 FROM ipam_ranges WHERE subnet = ?)
		`, ipRange, ipRange)
		if err != nil {
			return fmt.Errorf("error inserting IP range: %w", err)
		}
	}
	return nil
}

// getPodIPs obtains one Pod IP per IP family from the existing ipam ranges
func getIPRanges() ([]netip.Prefix, error) {
	// Query the database for all IP ranges and create allocators
	rows, err := db.Query("SELECT subnet FROM ipam_ranges")
	if err != nil {
		return nil, fmt.Errorf("error querying IP ranges: %w", err)
	}
	defer rows.Close()

	var cidrs []netip.Prefix
	for rows.Next() {
		var ipRange IPRange
		err := rows.Scan(&ipRange.Subnet)
		if err != nil {
			return nil, fmt.Errorf("error scanning IP range: %w", err)
		}
		// You might want to add validation here to ensure the subnet is valid
		cidr, err := netip.ParsePrefix(ipRange.Subnet)
		if err != nil {
			logger.Printf("invalid subnet CIDR %s : %v", ipRange.Subnet, err)
		} else {
			cidrs = append(cidrs, cidr)
		}
	}
	if len(cidrs) == 0 {
		return nil, fmt.Errorf("no ranges configured, can not allocate IPs for Pods")
	}
	return cidrs, nil
}

// IPAddress struct to hold both IPv4 and IPv6 addresses
type IPAddress struct {
	IPv4 sql.NullString
	IPv6 sql.NullString
}

func listIPAddresses() (ipv4 []string, ipv6 []string, err error) {
	// Query the database for IPv4 and IPv6 addresses
	rows, err := db.Query("SELECT ip_address_v4, ip_address_v6 FROM pods")
	if err != nil {
		return nil, nil, fmt.Errorf("error querying IP addresses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ipAddress IPAddress
		err := rows.Scan(&ipAddress.IPv4, &ipAddress.IPv6)
		if err != nil {
			return nil, nil, fmt.Errorf("error scanning IP address: %w", err)
		}
		if ipAddress.IPv4.Valid {
			ipv4 = append(ipv4, ipAddress.IPv4.String)
		}
		if ipAddress.IPv6.Valid {
			ipv6 = append(ipv6, ipAddress.IPv6.String)
		}
	}
	return
}

func getIPConfig(netconf *NetworkConfig) error {
	cidrs, err := getIPRanges()
	if err != nil {
		return err
	}

	for _, cidr := range cidrs {
		// skip ip families already allocated
		if cidr.Addr().Is4() && netconf.IPv4 != nil {
			continue
		}
		if cidr.Addr().Is6() && netconf.IPv6 != nil {
			continue
		}
		// Create an allocator
		alloc, err := NewAllocator(cidr)
		if err != nil {
			logger.Printf("can not allocate addresses from %s : %v", cidr.String(), err)
			continue
		}

		// This range is full try other
		if alloc.Free() == 0 {
			continue
		}

		ip, err := alloc.Allocate(netconf.ContainerID)
		if err != nil {
			logger.Printf("can not obtain addresses from %s : %v", cidr.String(), err)
			continue
		}

		if cidr.Addr().Is4() {
			netconf.IPv4 = net.IP(ip.AsSlice())
			netconf.GWv4 = net.IP(cidr.Masked().Addr().AsSlice())
		}

		if cidr.Addr().Is6() {
			netconf.IPv6 = net.IP(ip.AsSlice())
			netconf.GWv6 = net.IP(cidr.Masked().Addr().AsSlice())
		}
	}

	if netconf.IPv4 == nil && netconf.IPv6 == nil {
		return fmt.Errorf("no IPs available")
	}
	return nil
}

type Allocator struct {
	cidr     netip.Prefix
	ipFirst  netip.Addr
	ipLast   netip.Addr
	size     uint64
	reserved int // reserve first number of address
}

func NewAllocator(cidr netip.Prefix) (*Allocator, error) {
	var size uint64
	hostsBits := cidr.Addr().BitLen() - cidr.Bits()
	if hostsBits >= 64 {
		size = math.MaxInt64
	} else {
		size = uint64(1) << uint(hostsBits)
	}
	// skip the network address
	size = size - 1

	// leave some space free at the beginning since some environments
	// use those IPs to install well known servvices.
	reserved := 6
	if size <= 64 {
		reserved = 2
	} else if size <= 128 {
		reserved = 4
	}

	// Caching the first, offset and last addresses allows to optimize
	// the search loops by using the netip.Addr iterator instead
	// of having to do conversions with IP addresses.
	// Don't allocate the network's ".0" address.
	ipFirst := cidr.Masked().Addr().Next()
	// Don't allocate in the reserved zone
	ipFirst, err := addOffsetAddress(ipFirst, uint64(reserved))
	if err != nil {
		return nil, err
	}
	// Use the broadcast address as last address for IPv6
	ipLast, err := broadcastAddress(cidr)
	if err != nil {
		return nil, err
	}

	return &Allocator{
		cidr:     cidr,
		size:     size,
		reserved: reserved,
		ipFirst:  ipFirst,
		ipLast:   ipLast,
	}, nil
}

func ipExists(ip netip.Addr) (bool, error) {
	var count int
	var err error

	if ip.Is4() {
		err = db.QueryRow("SELECT COUNT(*) FROM pods WHERE ip_address_v4 = ?", ip.String()).Scan(&count)
	} else if ip.Is6() {
		err = db.QueryRow("SELECT COUNT(*) FROM pods WHERE ip_address_v6 = ?", ip.String()).Scan(&count)
	} else {
		return false, fmt.Errorf("invalid IP address type")
	}

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func ipInsert(ip netip.Addr, cidr netip.Prefix, id string) error {
	var result sql.Result
	var err error

	if cidr.Addr().Is4() {
		// Insert the container ID and IPv4 address into the database
		result, err = db.Exec(`
		UPDATE pods
		SET ip_address_v4 = ?, ip_gateway_v4 = ?
		WHERE container_id = ?
	`, ip.String(), cidr.Masked().Addr().String(), id)
	} else if cidr.Addr().Is6() {
		// Insert the container ID and IPv4 address into the database
		result, err = db.Exec(`
		UPDATE pods
		SET ip_address_v6 = ?, ip_gateway_v6 = ?
		WHERE container_id = ?
`, ip.String(), cidr.Masked().Addr().String(), id)
	}
	if err != nil {
		return fmt.Errorf("error updating container ID and IP %s : %v", cidr.String(), err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("update didn't succeed, rows affected %d", rowsAffected)
	}
	return nil
}

// IP iterator allows to iterate over all the IP addresses
// in a range defined by the start and last address.
// It starts iterating at the address position defined by the offset.
// It returns an invalid address to indicate it has finished.
func ipIterator(first netip.Addr, last netip.Addr, offset uint64) func() netip.Addr {
	// There are no modulo operations for IP addresses
	modulo := func(addr netip.Addr) netip.Addr {
		if addr.Compare(last) == 1 {
			return first
		}
		return addr
	}
	next := func(addr netip.Addr) netip.Addr {
		return modulo(addr.Next())
	}
	start, err := addOffsetAddress(first, offset)
	if err != nil {
		return func() netip.Addr { return netip.Addr{} }
	}
	start = modulo(start)
	ip := start
	seen := false
	return func() netip.Addr {
		value := ip
		// is the last or the first iteration
		if value == start {
			if seen {
				return netip.Addr{}
			}
			seen = true
		}
		ip = next(ip)
		return value
	}
}

func (a *Allocator) Allocate(id string) (netip.Addr, error) {
	rangeSize := a.size - uint64(a.reserved)
	var offset uint64
	switch {
	case rangeSize >= math.MaxInt64:
		offset = rand.Uint64()
	case rangeSize == 0:
		return netip.Addr{}, fmt.Errorf("not available addresses")
	default:
		offset = uint64(rand.Int63n(int64(rangeSize)))
	}

	iterator := ipIterator(a.ipFirst, a.ipLast, offset)
	for {
		ip := iterator()
		if !ip.IsValid() {
			break
		}
		// IP already exist
		ok, err := ipExists(ip)
		if err != nil {
			logger.Printf("error trying to check if ip %s exists : %v", ip.String(), err)
			continue
		}
		if ok {
			continue
		}

		err = ipInsert(ip, a.cidr, id)
		if err != nil {
			logger.Printf("error trying to store ip %s: %v", ip.String(), err)
			continue
		}
		return ip, nil

	}
	return netip.Addr{}, fmt.Errorf("allocator full")
}

func (a *Allocator) AllocateAddress(ip netip.Addr, id string) error {
	if !a.cidr.Contains(ip) {
		return fmt.Errorf("address %s out of range %s", ip.String(), a.cidr.String())
	}
	ok, err := ipExists(ip)
	if err != nil {
		logger.Printf("error trying to check if ip %s exists : %v", ip.String(), err)
		return err
	}
	if ok {
		return fmt.Errorf("address %s allready allocated", ip.String())
	}
	if a.ipFirst.Compare(ip) == 1 {
		return fmt.Errorf("address %s on the reserved space, lower than %s", ip.String(), a.ipFirst.String())
	}
	err = ipInsert(ip, a.cidr, id)
	if err != nil {
		logger.Printf("error trying to store ip %s: %v", ip.String(), err)
		return err
	}
	return nil
}

func (a *Allocator) Release(ip netip.Addr) {
	if ip.Is4() {
		_, _ = db.Exec("DELETE FROM pods WHERE ip_address_v4 = ?", ip.String())
	} else if ip.Is6() {
		_, _ = db.Exec("DELETE FROM pods WHERE ip_address_v6 = ?", ip.String())
	}
}

func (a *Allocator) Free() int {
	v4s, v6s, err := listIPAddresses()
	if err != nil {
		return 0
	}
	count := 0
	if a.cidr.Addr().Is4() {
		for _, ip := range v4s {
			addr, err := netip.ParseAddr(ip)
			if err != nil {
				continue
			}
			if a.cidr.Contains(addr) {
				count++
			}
		}
	} else if a.cidr.Addr().Is6() {
		for _, ip := range v6s {
			addr, err := netip.ParseAddr(ip)
			if err != nil {
				continue
			}
			if a.cidr.Contains(addr) {
				count++
			}
		}
	}
	return int(a.size) - count - a.reserved
}

// broadcastAddress returns the broadcast address of the subnet
// The broadcast address is obtained by setting all the host bits
// in a subnet to 1.
// network 192.168.0.0/24 : subnet bits 24 host bits 32 - 24 = 8
// broadcast address 192.168.0.255
func broadcastAddress(subnet netip.Prefix) (netip.Addr, error) {
	base := subnet.Masked().Addr()
	bytes := base.AsSlice()
	// get all the host bits from the subnet
	n := 8*len(bytes) - subnet.Bits()
	// set all the host bits to 1
	for i := len(bytes) - 1; i >= 0 && n > 0; i-- {
		if n >= 8 {
			bytes[i] = 0xff
			n -= 8
		} else {
			mask := ^uint8(0) >> (8 - n)
			bytes[i] |= mask
			break
		}
	}
	addr, ok := netip.AddrFromSlice(bytes)
	if !ok {
		return netip.Addr{}, fmt.Errorf("invalid address %v", bytes)
	}
	return addr, nil
}

// addOffsetAddress returns the address at the provided offset within the subnet
// TODO: move it to k8s.io/utils/net, this is the same as current AddIPOffset()
// but using netip.Addr instead of net.IP
func addOffsetAddress(address netip.Addr, offset uint64) (netip.Addr, error) {
	addressBytes := address.AsSlice()
	addressBig := big.NewInt(0).SetBytes(addressBytes)
	r := big.NewInt(0).Add(addressBig, big.NewInt(int64(offset))).Bytes()
	// r must be 4 or 16 bytes depending of the ip family
	// bigInt conversion to bytes will not take this into consideration
	// and drop the leading zeros, so we have to take this into account.
	lenDiff := len(addressBytes) - len(r)
	if lenDiff > 0 {
		r = append(make([]byte, lenDiff), r...)
	} else if lenDiff < 0 {
		return netip.Addr{}, fmt.Errorf("invalid address %v", r)
	}
	addr, ok := netip.AddrFromSlice(r)
	if !ok {
		return netip.Addr{}, fmt.Errorf("invalid address %v", r)
	}
	return addr, nil
}
