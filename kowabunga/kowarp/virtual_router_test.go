/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"fmt"
	"net"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestRouter(t *testing.T) {
	ip, _ := netlink.ParseAddr("127.0.0.34/24")
	ips := []netlink.Addr{*ip}
	itf, _ := net.InterfaceByIndex(2)
	iftUpAndRunning := itf.Flags&(net.FlagUp|net.FlagRunning) == (net.FlagUp | net.FlagRunning)
	if iftUpAndRunning {
		t.Errorf("Hello : \n %#v", itf.Flags.String())

	}
	vr, err := NewVirtualRouter(
		1,
		nil,
		254,
		ips,
		100,
		"",
		"",
		nil,
		false,
		[]VirtualRoute{},
	)
	if err != nil {
		t.Errorf("%v", err)
	}

	fmt.Println(vr.state)
}
