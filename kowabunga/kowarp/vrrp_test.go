/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"net"
	"testing"
	"time"

	"github.com/juju/errors"
)

// MUST BE RUN AS SUDO !!!!
func TestGeneratepacket(t *testing.T) {
	ip := []net.IP{
		{127, 0, 0, 1},
		{127, 0, 0, 1},
	}
	kwvrrp := &KowabungaVRRP{

		Version:      15,
		Type:         1,
		VirtualRtrID: 1,
		Priority:     255,
		CountIPAddr:  2,
		AdverInt:     3,
		Checksum:     0,
		IPAddresses:  ip,
	}

	conn, err := VRRPConn(ip[0], false)
	if err != nil {
		t.Errorf("connerr : %#v", err)
	}
	bytes, _ := kwvrrp.Serialize(&ip[0], &ip[1], true)
	if err != nil {
		t.Errorf("err serialize : %#v", err)
	}

	t.Errorf(" Serdata: \n%v\n", bytes)
	i, err := conn.WriteTo(bytes, &net.IPAddr{IP: ip[1]})
	if err != nil {
		t.Errorf("%d %#v", i, err)
	}
	buff := make([]byte, 1024)
	i, err = conn.Read(buff)
	if err != nil {
		t.Errorf("%d  %#v", i, err)
	}
	if err != nil {
		t.Fatalf("err set dead : %#v", err)
	}

	shittybuff := make([]byte, 1024)
	go func() {
		//conn.WriteTo(buff, &net.IPAddr{IP: ip[1]})
		i, err = conn.Read(shittybuff)

		t.Errorf("%s", err.Error())
	}()
	conn.Close()
	_, err = conn.File()
	if errors.Is(err, net.ErrClosed) {
		t.Errorf("Happy closed")
	}
	time.Sleep(5 * time.Second)
	// i, err = conn.Read(shittybuff)
	// if err != nil {
	// 	t.Errorf("%d  %#v", i, err)
	// }

	// Following commented only works if fd has been set to non blocking
	// err = conn.SetReadDeadline(time.Now().Add(5 * time.Microsecond))
	// i, err = conn.Read(buff)
	// if err != nil {
	// 	t.Errorf("%d  %#v", i, err)
	// }
	repvrrp := KowabungaVRRP{}
	hdrlen := int(buff[0]&0x0f) << 2
	err = repvrrp.DecodeFromBytes(buff[hdrlen:])
	if err != nil {
		t.Errorf(" %s,   %v", err.Error(), buff)
	}
	t.Errorf("sent %#v", kwvrrp)
	t.Errorf("received %#v", repvrrp)
	t.Errorf("Checksum received %b", repvrrp.Checksum)
	cstest, val, err := repvrrp.ValidateChecksum(&ip[0], &ip[1])
	if !cstest {
		t.Errorf("Sniff HECKSUM PA BO  %b", val)

	}

	// t.Errorf("Raw :  %v", buff)
	// t.Errorf(" %v", repvrrp)

	// var test uint8
	// test = 15<<4 + uint8(1)

	// t.Errorf("Version : %8b", test)
	// ver := test >> 4
	// typee := test << 4 >> 4
	// t.Errorf("Version : %d", ver)

	// t.Errorf("type : %d", typee)
}

func TestAddRemoveAddr(t *testing.T) {

	itf, err := findFirstPrivateInterface()
	if err != nil {
		t.Fatalf("err %s", err.Error())
	}
	t.Errorf("ITF : %s", itf.Name)

	ip, ipnet, _ := net.ParseCIDR("10.0.0.18/28")
	sz, _ := ipnet.Mask.Size()

	//sz, _ := net.IPv4Mask(ipnet.Mask[0], ipnet.Mask[1], ipnet.Mask[2], ipnet.Mask[3]).Size()
	t.Errorf("ip: %s , mask: %d", ip.String(), sz)

	// err = addIPToInterface(itf, &ip)
	// if err != nil {
	// 	t.Fatalf("err %s", err.Error())
	// }
	// err = removeIPFromInterface(itf, &ip)
	// if err != nil {
	// 	t.Fatalf("err %s", err.Error())
	// }
}
