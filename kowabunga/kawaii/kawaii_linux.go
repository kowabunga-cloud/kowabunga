/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kawaii

import (
	"fmt"
	"net"
	"slices"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"

	"github.com/vishvananda/netlink"
)

const (
	ErrNetlinkLinkAlreadyExist = "file exists"
	KawaiiIPsecMTU             = 1300
)

func SetXFRMInterfaces(metadata *metadata.InstanceMetadata, args ...any) error {
	handle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	for _, ipsec := range metadata.Kawaii.IPsecConnections {
		xfrmIndex := ipsec.XfrmId
		// Xfrmi
		xfrmAttr := netlink.NewLinkAttrs()
		xfrmAttr.MTU = KawaiiIPsecMTU
		xfrmAttr.Name = fmt.Sprintf("xfrm-%d", xfrmIndex)
		itf := &netlink.Xfrmi{
			Ifid:      uint32(xfrmIndex),
			LinkAttrs: xfrmAttr,
		}
		err := handle.LinkAdd(itf)
		if err != nil && err.Error() != ErrNetlinkLinkAlreadyExist {
			return fmt.Errorf("failed to create XFRM %d interface : %w", xfrmIndex, err)
		}
		err = handle.LinkSetUp(itf)
		if err != nil && err.Error() != ErrNetlinkLinkAlreadyExist {
			return fmt.Errorf("dailed to set XFRM %d interface Up : %w", xfrmIndex, err)
		}
		dest, err := netlink.ParseIPNet(ipsec.RemoteSubnet)
		if err != nil {
			return fmt.Errorf("failed to parse remote_subnet from metadata")
		}
		linkList, err := handle.LinkList()
		if err != nil {
			return err
		}
		xfrmLinkIndex := 0
		for _, l := range linkList {
			if l.Attrs().Name == fmt.Sprintf("xfrm-%d", xfrmIndex) {
				xfrmLinkIndex = l.Attrs().Index
				break
			}
		}

		route := &netlink.Route{
			LinkIndex: xfrmLinkIndex,
			Dst:       dest,
		}
		isVIPOwner, err := isVIPOwner(net.ParseIP(ipsec.IP))
		if err != nil {
			return err
		}
		routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		if isVIPOwner {
			err = removeConflictingRouteIfExists(route, routes)
			if err != nil {
				klog.Errorf("removeConflictingRouteIfExists: %s", err.Error())
			}
			err = handle.RouteAdd(route)
			if err != nil && err.Error() != ErrNetlinkLinkAlreadyExist {
				klog.Errorf("routeAdd: %s", err.Error())
			}
		} else {
			// if not owner of the IP, route must go to the IPsec owner:
			ip := findPrivateVIPIPsecPeerOwner(&ipsec, metadata.Kawaii)
			privateItfIndex, err := privateInterfaceIndex(metadata)
			if err != nil {
				return err
			}
			ipsecOwnerRoute := &netlink.Route{
				LinkIndex: privateItfIndex,
				Dst:       dest,
				Gw:        ip,
			}
			err = removeConflictingRouteIfExists(ipsecOwnerRoute, routes)
			if err != nil {
				klog.Errorf("removeConflictingRouteIfExists: %s", err.Error())
			}
			err = handle.RouteAdd(ipsecOwnerRoute)
			if err != nil && err.Error() != ErrNetlinkLinkAlreadyExist {
				klog.Errorf("routeAdd: %s", err.Error())
			}
		}
	}
	return nil
}

func RemoveXFRMInterfaces(metadata *metadata.InstanceMetadata, args ...any) error {
	handle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	links, err := handle.LinkList()
	if err != nil {
		return err
	}
	var xfrmIds []uint32
	for _, ipsec := range metadata.Kawaii.IPsecConnections {
		xfrmIds = append(xfrmIds, uint32(ipsec.XfrmId))
	}
	for _, l := range links {
		if l.Type() == "xfrm" {
			if !slices.Contains(xfrmIds, l.(*netlink.Xfrmi).Ifid) {
				routes, err := handle.RouteList(l, netlink.FAMILY_V4)
				if err != nil {
					return err
				}
				for _, r := range routes {
					err := handle.RouteDel(&r)
					if err != nil {
						klog.Errorf("Could not remove xfrm-%d route. Continuing...", l.(*netlink.Xfrmi).Ifid)
					}
				}
				err = handle.LinkDel(l)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func findPrivateVIPIPsecPeerOwner(ipsecConnMeta *metadata.KawaiiIPsecConnectionMetadata, kawaiiMetadata *metadata.KawaiiMetadata) net.IP {
	var matchingRouterID int
	matchingPrivateVIP := ""
	for _, VIPPeers := range kawaiiMetadata.VirtualIPs {
		if VIPPeers.VIP == ipsecConnMeta.IP {
			matchingRouterID = VIPPeers.VRRP
			break
		}
	}
	// Need to loop again against the same dataset
	// As our matching privateIP might have already been parsed in the loop
	for _, VIPPeers := range kawaiiMetadata.VirtualIPs {
		if VIPPeers.VIP != ipsecConnMeta.IP && VIPPeers.VRRP == matchingRouterID {
			matchingPrivateVIP = VIPPeers.VIP
			break
		}
	}
	return net.ParseIP(matchingPrivateVIP)
}

func privateInterfaceIndex(metadata *metadata.InstanceMetadata) (int, error) {
	handle, err := netlink.NewHandle()
	if err != nil {
		return -1, err
	}
	links, err := handle.LinkList()
	if err != nil {
		return -1, err
	}
	for _, l := range links {
		if l.Attrs().Name == metadata.Kawaii.PrivateInterface {
			return l.Attrs().Index, nil
		}
	}
	return -1, fmt.Errorf("could not find private interface index")
}

func isVIPOwner(ip net.IP) (bool, error) {
	handle, err := netlink.NewHandle()
	if err != nil {
		return false, err
	}
	addrs, err := handle.AddrList(nil, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}
	for _, a := range addrs {
		if a.IP.Equal(ip) {
			return true, nil
		}
	}
	return false, nil
}

func removeConflictingRouteIfExists(route *netlink.Route, routes []netlink.Route) error {
	handle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	for _, r := range routes {
		if route.Dst.IP.Equal(r.Dst.IP) &&
			route.Dst.Mask.String() == r.Dst.Mask.String() &&
			(!route.Gw.Equal(r.Gw) || route.LinkIndex != r.LinkIndex) {
			klog.Infof("Updating route to %s", r.Dst.String())
			err := handle.RouteDel(&r)
			return err
		}
	}
	return nil
}
