/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"fmt"
	"net"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/vishvananda/netlink"
)

// Finds the first private, non-loopback, running interface on the host
func findFirstPrivateInterface() (*net.Interface, error) {
	var ipv4Addr net.IP

	hasPrivateIPv4 := false
	itfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, itf := range itfs {
		iftUpAndRunning := itf.Flags&(net.FlagUp|net.FlagRunning) == (net.FlagUp | net.FlagRunning)
		isLoopback := itf.Flags&net.FlagLoopback == 1
		addrs, err := itf.Addrs()
		if err != nil {
			return nil, fmt.Errorf("interface do not have any IP assigned")
		}
		for _, addr := range addrs { // get ipv4 address
			if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil && ipv4Addr.IsPrivate() {
				hasPrivateIPv4 = true
			}
		}
		if itf.HardwareAddr != nil && iftUpAndRunning && !isLoopback && hasPrivateIPv4 {
			return &itf, nil
		}
	}
	return nil, fmt.Errorf("could not find a running, non-loopback, with a private IPV4 assigned interface running")
}

func ipLargerThan(ip1, ip2 net.IP) bool {
	if len(ip1) != len(ip2) {
		klog.Errorf("virtual router: comparing IPs require them to have the same size. ip1 : %d, ip2 : %d", len(ip1), len(ip2))
	}
	for i := range ip1 {
		if ip1[i] > ip2[i] {
			return true
		} else if ip1[i] < ip2[i] {
			return false
		}
	}
	return false
}

func findIPbyInterface(itf *net.Interface) (net.IP, error) {
	var addrs, errOfListAddrs = itf.Addrs()
	if errOfListAddrs != nil {
		return nil, fmt.Errorf("findIPbyInterface: %v", errOfListAddrs)
	}
	for index := range addrs {
		var ipaddr, _, errOfParseIP = net.ParseCIDR(addrs[index].String())
		if errOfParseIP != nil {
			return nil, fmt.Errorf("findIPbyInterface: %v", errOfParseIP)
		}
		if ipaddr.To4() != nil {
			if ipaddr.IsGlobalUnicast() {
				return ipaddr, nil
			}
		}
	}
	return nil, fmt.Errorf("findIPbyInterface: can not find valid IP addrs on %v", itf.Name)
}

func addIPToInterface(itf *net.Interface, ip *netlink.Addr) error {
	advEth, err := netlink.LinkByName(itf.Name)
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(advEth, ip)
	if err != nil {
		return err
	}
	return nil
}

func removeIPFromInterface(itf *net.Interface, ip *netlink.Addr) error {
	advEth, err := netlink.LinkByName(itf.Name)
	if err != nil {
		return err
	}
	err = netlink.AddrDel(advEth, ip)
	if err != nil {
		return err
	}
	return nil
}

func addRoute(itf *net.Interface, via *net.IP, dst *net.IPNet, metric int) error {
	link, err := netlink.LinkByName(itf.Name)
	if err != nil {
		return err
	}
	route := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Gw: *via, Priority: metric}
	err = netlink.RouteAdd(&route)
	if err != nil {
		return err
	}
	return nil
}

func removeRoute(itf *net.Interface, via *net.IP, dst *net.IPNet, metric int) error {
	link, err := netlink.LinkByName(itf.Name)
	if err != nil {
		return err
	}
	route := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Gw: *via, Priority: metric}
	err = netlink.RouteDel(&route)
	if err != nil {
		return err
	}
	return nil
}

func ipListsEqual(l1 []net.IP, l2 []net.IP) bool {
	for _, ip1 := range l1 {
		peerExists := false
		for _, ip2 := range l2 {
			if ip1.Equal(ip2) {
				peerExists = true
				break
			}
		}
		if !peerExists {
			return false
		}
	}
	return true
}

// Same ft, different interface, can't merge :(
func ipAddrListsEqual(l1 []netlink.Addr, l2 []netlink.Addr) bool {
	for _, ip1 := range l1 {
		peerExists := false
		for _, ip2 := range l2 {
			if ip1.Equal(ip2) {
				peerExists = true
				break
			}
		}
		if !peerExists {
			return false
		}
	}
	return true
}

// Same ft, different interface, can't merge :(
func ipVirtualRoutesEqual(l1 []VirtualRoute, l2 []VirtualRoute) bool {
	for _, route1 := range l1 {
		peerExists := false
		for _, route2 := range l2 {
			if route1.Equal(&route2) {
				peerExists = true
				break
			}
		}
		if !peerExists {
			return false
		}
	}
	return true
}
