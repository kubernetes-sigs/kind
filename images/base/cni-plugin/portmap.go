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
	"bytes"
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var protoMap map[string]byte = map[string]byte{
	"tcp":  unix.IPPROTO_TCP,
	"udp":  unix.IPPROTO_UDP,
	"sctp": unix.IPPROTO_SCTP,
}

func getPortMapEntries() ([]PortMapConfig, error) {
	rows, err := db.Query(`
		SELECT host_ip, host_port, protocol, container_ip, container_port 
		FROM portmap_entries
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying port map entries: %w", err)
	}
	defer rows.Close()

	var entries []PortMapConfig
	for rows.Next() {
		var entry PortMapConfig
		err := rows.Scan(
			&entry.HostIP, &entry.HostPort, &entry.Protocol,
			&entry.ContainerIP, &entry.ContainerPort,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning port map entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// reconcilePortMaps gets all the existing portmaps from the database
// and reconfigures all the nftables with the db state
func reconcilePortMaps() error {
	entries, err := getPortMapEntries()
	if err != nil {
		return err
	}

	nft, err := nftables.New()
	if err != nil {
		return fmt.Errorf("portmap failure, can not start nftables:%v", err)
	}

	portmapTable := nft.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   pluginName,
	})
	nft.FlushTable(portmapTable)

	// Sets
	hostPortMapv4Set := &nftables.Set{
		Table:         portmapTable,
		Name:          hostPortMapv4,
		IsMap:         true,
		Interval:      true,
		Concatenation: true,
		KeyType:       nftables.MustConcatSetType(nftables.TypeIPAddr, nftables.TypeInetProto, nftables.TypeInetService),
		DataType:      nftables.MustConcatSetType(nftables.TypeIPAddr, nftables.TypeInetService),
	}

	hostPortMapv6Set := &nftables.Set{
		Table:         portmapTable,
		Name:          hostPortMapv6,
		IsMap:         true,
		Interval:      true,
		Concatenation: true,
		KeyType:       nftables.MustConcatSetType(nftables.TypeIP6Addr, nftables.TypeInetProto, nftables.TypeInetService),
		DataType:      nftables.MustConcatSetType(nftables.TypeIP6Addr, nftables.TypeInetService),
	}

	if err := nft.AddSet(hostPortMapv4Set, nil); err != nil {
		return fmt.Errorf("failed to add Set %s : %v", hostPortMapv4Set.Name, err)
	}
	nft.FlushSet(hostPortMapv4Set)

	if err := nft.AddSet(hostPortMapv6Set, nil); err != nil {
		return fmt.Errorf("failed to add Set %s : %v", hostPortMapv6Set.Name, err)
	}
	nft.FlushSet(hostPortMapv6Set)

	// Set Elements
	var elementsV4, elementsV6 []nftables.SetElement
	for _, e := range entries {
		hostIP, err := netip.ParseAddr(e.HostIP)
		if err != nil {
			logger.Printf("could not parse HostIP %s : %v", e.HostIP, err)
			continue
		}
		containerIP, err := netip.ParseAddr(e.ContainerIP)
		if err != nil {
			logger.Printf("could not parse ContainerIP %s : %v", e.ContainerIP, err)
			continue
		}

		// NFT datatypes https://github.com/google/nftables/blob/e99829fb4f26d75fdd0cfce8ba4632744e72c2bc/set.go#L71
		// TypeInetProto 1 byte
		// TypeInetService 2 bytes
		// Netlink seems to require 4-byte alignment.
		key := []byte{}

		key = append(key, hostIP.AsSlice()...)
		proto := protoMap[strings.ToLower(e.Protocol)]
		key = append(key, encodeWithAlignment(proto)...)
		key = append(key, encodeWithAlignment(uint16(e.HostPort))...)

		val := []byte{}
		val = append(val, containerIP.AsSlice()...)
		val = append(val, encodeWithAlignment(uint16(e.ContainerPort))...)

		element := nftables.SetElement{
			Key: key,
			Val: val,
		}
		// from all zeros to all ones to cover any IP address
		if hostIP.IsUnspecified() {
			keyEnd := make([]byte, len(key))
			_ = copy(keyEnd, key)
			for i := 0; i < len(hostIP.AsSlice()); i++ {
				keyEnd[i] = 0xff
			}
			element.KeyEnd = keyEnd
		}
		if containerIP.Is4() {
			elementsV4 = append(elementsV4, element)
		} else if containerIP.Is6() {
			elementsV6 = append(elementsV6, element)
		}
	}

	if err := nft.SetAddElements(hostPortMapv4Set, elementsV4); err != nil {
		return fmt.Errorf("failed to add Set %s : %v", hostPortMapv4Set.Name, err)
	}
	if err := nft.SetAddElements(hostPortMapv6Set, elementsV6); err != nil {
		return fmt.Errorf("failed to add Set %s : %v", hostPortMapv6Set.Name, err)
	}

	// Add chains
	preroutingChain := nft.AddChain(&nftables.Chain{
		Name:     "prerouting",
		Table:    portmapTable,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityNATDest,
	})

	/*
			nft --debug=netlink add rule inet cni-kindnet prerouting dnat ip to ip daddr . ip protocol . th dport map @hostport-map-v4
		inet cni-kindnet prerouting
		  [ meta load nfproto => reg 1 ]
		  [ cmp eq reg 1 0x00000002 ]
		  [ payload load 4b @ network header + 16 => reg 1 ]
		  [ payload load 1b @ network header + 9 => reg 9 ]
		  [ payload load 2b @ transport header + 2 => reg 10 ]
		  [ lookup reg 1 set hostport-map-v4 dreg 1 ]
		  [ nat dnat ip addr_min reg 1 proto_min reg 9 ]
	*/

	mapLookupV4Expressions := []expr.Any{
		// only packets destined to local addresses
		&expr.Fib{
			Register:       1,
			FlagDADDR:      true,
			ResultADDRTYPE: true,
		},
		&expr.Cmp{
			// [ cmp eq reg 1 0x00000002 ]
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     encodeWithAlignment(byte(unix.RTN_LOCAL)),
		},
		&expr.Meta{
			// [ meta load nfproto => reg 1 ]
			Key:      expr.MetaKeyNFPROTO,
			Register: 1,
		},
		&expr.Cmp{
			// [ cmp eq reg 1 0x00000002 ]
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.NFPROTO_IPV4},
		},
		&expr.Payload{
			// [ payload load 4b @ network header + 16 => reg 1 ]
			DestRegister: 1,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       16,
			Len:          4,
		},
		&expr.Payload{
			// [ payload load 1b @ network header + 9 => reg 9 ]
			DestRegister: 9,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       9,
			Len:          1,
		},
		// [ payload load 2b @ transport header + 2 => reg 10 ]
		&expr.Payload{
			DestRegister: 10,
			Base:         expr.PayloadBaseTransportHeader,
			Offset:       2,
			Len:          2,
		},
		&expr.Lookup{
			// [ lookup reg 1 set hostport-map-v4 dreg 1 ]
			SourceRegister: 1,
			DestRegister:   1,
			IsDestRegSet:   true,
			SetName:        hostPortMapv4Set.Name,
		},
		&expr.NAT{
			// [ nat dnat ip addr_min reg 1 proto_min reg 9 ]
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV4,
			RegAddrMin:  1,
			RegProtoMin: 9,
			Specified:   true,
		},
	}

	mapLookupV6Expressions := []expr.Any{
		// only packets destined to local addresses
		&expr.Fib{
			Register:       1,
			FlagDADDR:      true,
			ResultADDRTYPE: true,
		},
		&expr.Cmp{
			// [ cmp eq reg 1 0x00000002 ]
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     encodeWithAlignment(byte(unix.RTN_LOCAL)),
		},
		&expr.Meta{
			// [ meta load nfproto => reg 1 ]
			Key:      expr.MetaKeyNFPROTO,
			Register: 1,
		},
		&expr.Cmp{
			// [ cmp eq reg 1 0x0000000a ]
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.NFPROTO_IPV6},
		},
		&expr.Payload{
			// [ payload load 16b @ network header + 24 => reg 1 ]
			DestRegister: 1,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       24,
			Len:          16,
		},
		&expr.Meta{
			// [ meta load l4proto => reg 2 ]
			Key:      expr.MetaKeyL4PROTO,
			Register: 2,
		},
		// [ payload load 2b @ transport header + 2 => reg 13 ]
		&expr.Payload{
			DestRegister: 13,
			Base:         expr.PayloadBaseTransportHeader,
			Offset:       2,
			Len:          2,
		},
		&expr.Lookup{
			// [ lookup reg 1 set hostport-map-v6 dreg 1 ]
			SourceRegister: 1,
			DestRegister:   1,
			IsDestRegSet:   true,
			SetName:        hostPortMapv6Set.Name,
		},
		&expr.NAT{
			// [ nat dnat ip addr_min reg 1 proto_min reg 9 ]
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV6,
			RegAddrMin:  1,
			RegProtoMin: 2,
			Specified:   true,
		},
	}

	// IPv4
	nft.AddRule(&nftables.Rule{
		Table: portmapTable,
		Chain: preroutingChain,
		Exprs: mapLookupV4Expressions,
	})

	// IPv6
	nft.AddRule(&nftables.Rule{
		Table: portmapTable,
		Chain: preroutingChain,
		Exprs: mapLookupV6Expressions,
	})

	output := nft.AddChain(&nftables.Chain{
		Name:     "output",
		Table:    portmapTable,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityNATDest,
	})

	// IPv4
	nft.AddRule(&nftables.Rule{
		Table: portmapTable,
		Chain: output,
		Exprs: mapLookupV4Expressions,
	})

	// IPv6
	nft.AddRule(&nftables.Rule{
		Table: portmapTable,
		Chain: output,
		Exprs: mapLookupV6Expressions,
	})

	err = nft.Flush()
	if err != nil {
		return fmt.Errorf("failed to add nftables for portmaps: %v", err)
	}
	return nil
}

func encodeWithAlignment(data any) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}

	// Calculate padding
	padding := (4 - buf.Len()%4) % 4
	for i := 0; i < padding; i++ {
		buf.WriteByte(0x00)
	}

	return buf.Bytes()
}

// deletePortmapStaleConnections delete the UDP conntrack entries on the specified IP family
// from the ports mapped to the container
func deletePortmapStaleConnections(portMappings []PortMapConfig) error {
	filtersV4 := []netlink.CustomConntrackFilter{}
	filtersV6 := []netlink.CustomConntrackFilter{}
	for _, pm := range portMappings {
		// skip if is not UDP
		if strings.ToLower(pm.Protocol) != "udp" {
			continue
		}
		filter := &netlink.ConntrackFilter{}
		filter.AddProtocol(unix.IPPROTO_UDP)
		filter.AddPort(netlink.ConntrackOrigDstPort, uint16(pm.HostPort))
		ip, err := netip.ParseAddr(pm.ContainerIP)
		if err != nil {
			continue
		}
		if ip.Is4() {
			filtersV4 = append(filtersV4, filter)
		} else if ip.Is6() {
			filtersV6 = append(filtersV6, filter)
		}
	}
	if len(filtersV4) > 0 {
		_, err := netlink.ConntrackDeleteFilters(netlink.ConntrackTable, unix.AF_INET, filtersV4...)
		if err != nil {
			logger.Printf("error deleting connection tracking state error for IPv4: %v", err)
		}
	}
	if len(filtersV6) > 0 {
		_, err := netlink.ConntrackDeleteFilters(netlink.ConntrackTable, unix.AF_INET6, filtersV6...)
		if err != nil {
			logger.Printf("error deleting connection tracking state error for IPv6: %v", err)
		}
	}
	return nil
}
