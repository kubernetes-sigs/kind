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
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"

	"golang.org/x/sys/unix"
)

func createPodInterface(netconf *NetworkConfig) error {
	iifName := defaultInterface      // veth interface name in the container
	oofName := netconf.InterfaceName // veth interface name in the host

	// Get network namespace file descriptors
	rootNs, err := netns.Get()
	if err != nil {
		return err
	}
	defer rootNs.Close()

	containerNs, err := netns.GetFromPath(netconf.NetNS)
	if err != nil {
		return fmt.Errorf("could not get network namespace from path %s : %w", netconf.NetNS, err)
	}

	// Create veth interface, internal name is eth0 and external is randomly generated
	// with the knet prefix.
	flags := unix.NLM_F_CREATE | unix.NLM_F_EXCL | unix.NLM_F_ACK
	req := nl.NewNetlinkRequest(unix.RTM_NEWLINK, flags)
	// Get a netlink socket in the container namespace namespace
	s, err := nl.GetNetlinkSocketAt(containerNs, rootNs, unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("could not get network namespace handle: %w", err)
	}
	req.Sockets = map[int]*nl.SocketHandle{
		unix.NETLINK_ROUTE: {Socket: s},
	}

	msg := nl.NewIfInfomsg(unix.AF_UNSPEC)
	req.AddData(msg)

	// internal interface name is always eth0
	nameData := nl.NewRtAttr(unix.IFLA_IFNAME, nl.ZeroTerminated(iifName))
	req.AddData(nameData)

	// base namespace is the container namespace
	val := nl.Uint32Attr(uint32(containerNs))
	attr := nl.NewRtAttr(unix.IFLA_NET_NS_FD, val)
	req.AddData(attr)

	// the link type is veth
	linkInfo := nl.NewRtAttr(unix.IFLA_LINKINFO, nil)
	linkInfo.AddRtAttr(nl.IFLA_INFO_KIND, nl.NonZeroTerminated("veth"))

	// veth interface on the root namespace
	data := linkInfo.AddRtAttr(nl.IFLA_INFO_DATA, nil)
	peer := data.AddRtAttr(nl.VETH_INFO_PEER, nil)
	nl.NewIfInfomsgChild(peer, unix.AF_UNSPEC)
	peer.AddRtAttr(unix.IFLA_IFNAME, nl.ZeroTerminated(oofName))

	valRoot := nl.Uint32Attr(uint32(rootNs))
	peer.AddRtAttr(unix.IFLA_NET_NS_FD, valRoot)

	req.AddData(linkInfo)

	// execute the netlink operation to create the veth
	_, err = req.Execute(unix.NETLINK_ROUTE, 0)
	if err != nil {
		return fmt.Errorf("fail to add interface on namespace %s : %v", netconf.NetNS, err)
	}

	// veth interface has been created, add routes and addresses now

	// best effort to set the loopback interface up
	// open a netlink handle on the container namespace to use it directly
	// and avoid running goroutines inside network namespaces that does not
	// work fine and require to lock the entire thread.
	nhNs, err := netlink.NewHandleAt(containerNs)
	if err != nil {
		return err
	}
	loLink, err := nhNs.LinkByName("lo")
	if err != nil && !errors.Is(err, unix.EINTR) {
		return fmt.Errorf("could not get interface loopback on namespace %s : %w", netconf.NetNS, err)
	}
	_ = nhNs.LinkSetUp(loLink)

	// set up the veth inside the container
	nsLink, err := nhNs.LinkByName(iifName)
	if err != nil && !errors.Is(err, unix.EINTR) {
		return fmt.Errorf("could not get interface %s on namespace %s : %w", iifName, netconf.NetNS, err)
	}

	// set MTU if defined
	if netconf.MTU > 0 {
		err = nhNs.LinkSetMTU(nsLink, netconf.MTU)
		if err != nil {
			return fmt.Errorf("could not set interface %s mtu : %w", iifName, err)
		}
	}

	if err = nhNs.LinkSetUp(nsLink); err != nil {
		return fmt.Errorf("failed to set interface %s up: %v", iifName, err)
	}

	// DAD has to be disabled to avoid latency on network start
	if netconf.IPv6 != nil {
		// set up the veth inside the host outside of the container
		// don't accept Router Advertisements
		_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_ra", oofName), []byte(strconv.Itoa(0)), 0640)
		// DAD increases pod latency and we guarantee IP uniqueness
		_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_dad", oofName), []byte(strconv.Itoa(0)), 0640)
		_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/dad_transmits", oofName), []byte(strconv.Itoa(0)), 0640)

		// set sysctls inside the container
		err = func() error {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			err = netns.Set(containerNs)
			if err != nil {
				return err
			}
			_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_ra", iifName), []byte(strconv.Itoa(0)), 0640)
			// DAD increases pod latency and we guarantee IP uniqueness
			_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_dad", iifName), []byte(strconv.Itoa(0)), 0640)
			_ = os.WriteFile(fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/dad_transmits", iifName), []byte(strconv.Itoa(0)), 0640)

			netns.Set(rootNs)
			return nil
		}()
		if err != nil {
			return fmt.Errorf("error setting IPv6 sysctl inside the container")
		}
	}

	nhRoot, err := netlink.NewHandle()
	if err != nil {
		return fmt.Errorf("could not get netlink handle on host : %w", err)
	}

	hostLink, err := nhRoot.LinkByName(oofName)
	if err != nil && !errors.Is(err, unix.EINTR) {
		return fmt.Errorf("could not get interface %s on host : %w", oofName, err)
	}

	// set metadata on the alias field with the pod name and namespace
	if netconf.Name != "" {
		err = nhRoot.LinkSetAlias(hostLink, fmt.Sprintf("link-pod %s/%s", netconf.Namespace, netconf.Name))
		if err != nil {
			logger.Printf("could not set interface %s alias : %v", oofName, err)
		}
	}

	// set MTU if defined
	if netconf.MTU > 0 {
		err = nhRoot.LinkSetMTU(hostLink, netconf.MTU)
		if err != nil {
			return fmt.Errorf("could not set interface %s mtu : %w", oofName, err)
		}
	}

	if err = nhRoot.LinkSetUp(hostLink); err != nil {
		return fmt.Errorf("failed to set interface %s up: %v", oofName, err)
	}

	// Configure addresses and routes
	if netconf.IPv4 != nil {
		// add the Pod address as a /32 inside the namespace and use the "onlink" flag on the
		// default gateway to avoid arp. The default gateway is added to the lo interface on the
		// veth host.
		address := &netlink.Addr{IPNet: &net.IPNet{IP: netconf.IPv4, Mask: net.CIDRMask(32, 32)}}
		err = nhNs.AddrAdd(nsLink, address)
		if err != nil {
			return fmt.Errorf("could not add address %s on namespace %s : %w", netconf.IPv4.String(), netconf.NetNS, err)
		}

		addressGw := &netlink.Addr{IPNet: &net.IPNet{IP: netconf.GWv4, Mask: net.CIDRMask(32, 32)}}
		err = netlink.AddrAdd(hostLink, addressGw)
		if err != nil {
			return fmt.Errorf("could not add address %s on interface %s : %w", netconf.IPv4.String(), oofName, err)
		}

		// set the route from the network namespace to the host
		routeToGateway := netlink.Route{
			LinkIndex: nsLink.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       addressGw.IPNet,
		}

		// TODO: investigate how to make this more resilient
		// CNI ADD command failed: fail to create veth interface: could not add route {Ifindex: 43656 Dst: 2001:db8::7775:5443:1309:8685/128 Src: 2001:db8:: Gw: <nil> Flags: [] Table: 0 Realm: 0} on the host to the container interface knetb3773c91 : invalid argument
		time.Sleep(10 * time.Millisecond)
		if err := nhNs.RouteAdd(&routeToGateway); err != nil {
			return fmt.Errorf("could not add route to default gw on namespace %s : %w", netconf.NetNS, err)
		}

		// set the route from the network namespace to the host
		defaultRoute := netlink.Route{
			LinkIndex: nsLink.Attrs().Index,
			Src:       netconf.IPv4,
			Gw:        netconf.GWv4,
			Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 0)},
		}
		if err := nhNs.RouteAdd(&defaultRoute); err != nil {
			return fmt.Errorf("could not add default route on namespace %s : %w", netconf.NetNS, err)
		}

		// set the route from the host to the network namespace
		route := netlink.Route{
			LinkIndex: hostLink.Attrs().Index,
			Src:       netconf.GWv4,
			Scope:     netlink.SCOPE_HOST,
			Dst:       address.IPNet,
		}
		if err := nhRoot.RouteAdd(&route); err != nil {
			return fmt.Errorf("could not add route %v on host to the container interface %s : %w", route, hostLink.Attrs().Name, err)
		}
	}

	if netconf.IPv6 != nil {
		// add the Pod address as a /128 inside the namespace and use the "onlink" flag on the
		// default gateway to avoid discovery. The default gateway is added to the veth host interface.
		address := &netlink.Addr{
			IPNet: &net.IPNet{IP: netconf.IPv6, Mask: net.CIDRMask(128, 128)},
			Flags: unix.IFA_F_PERMANENT,
		}
		err = nhNs.AddrAdd(nsLink, address)
		if err != nil {
			return fmt.Errorf("could not add address %s on namespace %s : %w", netconf.IPv6.String(), netconf.NetNS, err)
		}

		addressGw := &netlink.Addr{IPNet: &net.IPNet{IP: netconf.GWv6, Mask: net.CIDRMask(128, 128)}}
		err = nhRoot.AddrAdd(hostLink, addressGw)
		if err != nil {
			return fmt.Errorf("could not add address %s on interface %s : %w", netconf.GWv6.String(), oofName, err)
		}

		// set the route from the network namespace to the host
		routeToGateway := netlink.Route{
			LinkIndex: nsLink.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       addressGw.IPNet,
		}

		// TODO: investigate how to make this more resilient
		// CNI ADD command failed: fail to create veth interface: could not add route {Ifindex: 43656 Dst: 2001:db8::7775:5443:1309:8685/128 Src: 2001:db8:: Gw: <nil> Flags: [] Table: 0 Realm: 0} on the host to the container interface knetb3773c91 : invalid argument
		time.Sleep(10 * time.Millisecond)
		if err := nhNs.RouteAdd(&routeToGateway); err != nil {
			return fmt.Errorf("could not add route to default gw on namespace %s : %w", netconf.NetNS, err)
		}

		// set the route from the network namespace to the host
		defaultRoute := netlink.Route{
			LinkIndex: nsLink.Attrs().Index,
			Gw:        netconf.GWv6,
			Dst:       &net.IPNet{IP: net.IPv6zero, Mask: net.CIDRMask(0, 0)},
		}

		if err := nhNs.RouteAdd(&defaultRoute); err != nil {
			return fmt.Errorf("could not add default route on namespace %s : %w", netconf.NetNS, err)
		}

		route := netlink.Route{
			LinkIndex: hostLink.Attrs().Index,
			Src:       netconf.GWv6,
			Scope:     netlink.SCOPE_HOST,
			Dst:       address.IPNet,
		}
		if err := nhRoot.RouteAdd(&route); err != nil {
			return fmt.Errorf("could not add route %v on the host to the container interface %s : %w", route, hostLink.Attrs().Name, err)
		}
	}
	return nil
}

func deletePodInterface(ifName string, netNS string) error {
	containerNs, err := netns.GetFromPath(netNS)
	if err != nil {
		return fmt.Errorf("could not get network namespace from path %s for network device %s : %w", netNS, ifName, err)
	}
	// to avoid golang problem with goroutines we create the socket in the
	// namespace and use it directly
	nhNs, err := netlink.NewHandleAt(containerNs)
	if err != nil {
		// namespace no longer exist
		return nil
	}

	nsLink, err := nhNs.LinkByName(ifName)
	if err != nil && !errors.Is(err, unix.EINTR) {
		// interface is no present so no need to delete
		return nil
	}

	err = nhNs.LinkDel(nsLink)
	if err != nil {
		return fmt.Errorf("failed to delete %q: %v", ifName, err)
	}
	return nil
}

func getDefaultGwInterfaceMTU() int {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil && !errors.Is(err, unix.EINTR) {
		return 0
	}

	for _, r := range routes {
		if r.Dst.IP.Equal(net.IPv4zero) || r.Dst.IP.Equal(net.IPv6zero) {
			intfLink, err := netlink.LinkByIndex(r.LinkIndex)
			if err != nil {
				logger.Printf("Failed to get interface link for route %v : %v", r, err)
				continue
			}
			return intfLink.Attrs().MTU
		}
	}
	return 0
}
