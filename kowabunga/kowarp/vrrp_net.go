/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

// Only 1 vrrpNet object shall exist during runtime
import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/mdlayher/arp"
)

const (
	BroadcastAddress              = "ff:ff:ff:ff:ff:ff"
	KowabungaVRRPMulticastAddress = "224.0.0.18"
)

type vrrpConn struct {
	conn     *net.IPConn
	refCount uint8 // No more than 255 routers possible, so there can't be more than 255 refs
}

type vrrpNet struct {
	multicastReader *net.IPConn
	multicastWriter *net.IPConn
	unicastConn     map[string]*vrrpConn // string must be an sourceIP
	arpClient       *arp.Client
}

func (vrrpNet *vrrpNet) addRouter(vr *virtualRouter) error {
	// Multicast is used if no peers are configured.
	// Channel should already be opened on object init
	klog.Infof("Prepare connections for virtual router %d, %s", vr.ID, vr.PreferredSourceIP.To4().String())
	var err error
	if len(vr.Peers) == 0 {
		if vrrpNet.multicastWriter == nil {
			vrrpNet.multicastWriter, err = VRRPConn(vr.PreferredSourceIP, true)
		}
		if vrrpNet.multicastReader == nil {
			vrrpNet.multicastReader, err = VRRPMulticastReaderConn(net.ParseIP(KowabungaVRRPMulticastAddress), vr.PreferredSourceIP)
		}
	} else {
		if vrrpNet.unicastConn[vr.PreferredSourceIP.String()] == nil {
			sender, err := VRRPConn(vr.PreferredSourceIP, false)
			if err != nil {
				return err
			}
			vrrpConn := &vrrpConn{sender, 0}
			vrrpNet.unicastConn[vr.PreferredSourceIP.String()] = vrrpConn
			go vrrpNet.PollConn(vrrpNet.unicastConn[vr.PreferredSourceIP.String()].conn)
		}
		vrrpNet.unicastConn[vr.PreferredSourceIP.String()].refCount++
	}
	return err
}

func (vrrpNet *vrrpNet) removeRouter(vr *virtualRouter) error {
	var err error
	if len(vr.Peers) != 0 {
		vrrpNet.unicastConn[vr.PreferredSourceIP.String()].refCount--
		if vrrpNet.unicastConn[vr.PreferredSourceIP.String()].refCount == 0 {
			err := vrrpNet.unicastConn[vr.PreferredSourceIP.String()].conn.Close()
			if err != nil {
				return err
			}
			delete(vrrpNet.unicastConn, vr.PreferredSourceIP.String())
		}
	}
	return err
}

func (vrrpNet *vrrpNet) Shutdown() error {
	var err error
	err = vrrpNet.multicastReader.Close()
	if err != nil {
		return err
	}
	err = vrrpNet.multicastWriter.Close()
	if err != nil {
		return err
	}
	for _, w := range vrrpNet.unicastConn {
		err = w.conn.Close()
		if err != nil {
			return err
		}
	}
	err = vrrpNet.arpClient.Close()
	return err
}

func VRRPMulticastReaderConn(multicastAddress, local net.IP) (*net.IPConn, error) {
	conn, err := net.ListenIP("ip4:112", &net.IPAddr{IP: multicastAddress})
	if err != nil {
		return nil, err
	}
	var fd, errOfGetFD = conn.File()
	if errOfGetFD != nil {
		return nil, errOfGetFD
	}
	defer func() {
		_ = fd.Close()
	}()

	var mreq = &syscall.IPMreq{
		Multiaddr: [4]byte(multicastAddress.To4()),
		Interface: [4]byte(local.To4()),
	}
	if errSetMreq := syscall.SetsockoptIPMreq(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq); errSetMreq != nil {
		return nil, fmt.Errorf("VRRPMulticastReaderConn: %v", errSetMreq)
	}
	return nil, nil
}

// Prepare Multicast OR Unicast
func VRRPConn(src net.IP, isForMulticastTarget bool) (*net.IPConn, error) {

	conn, err := net.ListenIP("ip4:112", &net.IPAddr{IP: src})
	if err != nil {
		return nil, err
	}
	var fd, errOfGetFD = conn.File()
	if errOfGetFD != nil {
		return nil, errOfGetFD
	}
	defer func() {
		_ = fd.Close()
	}()
	//Multicast
	if isForMulticastTarget {
		//set TTL
		if errOfSetTTL := syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_MULTICAST_TTL, int(KowabungaVRRPTTL)); errOfSetTTL != nil {
			return nil, fmt.Errorf("VRRPConn: %v", errOfSetTTL)
		}
		//disable multicast loop
		if errOfSetLoop := syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 0); errOfSetLoop != nil {
			return nil, fmt.Errorf("VRRPConn: %v", errOfSetLoop)
		}
	} else {
		//Unicast
		if errOfSetTTL := syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_TTL, int(KowabungaVRRPTTL)); errOfSetTTL != nil {
			return nil, fmt.Errorf("VRRPConn: %v", errOfSetTTL)
		}
	}
	//set tos
	if errOfSetTOS := syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_TOS, 7); errOfSetTOS != nil {
		return nil, fmt.Errorf("VRRPConn: %v", errOfSetTOS)
	}

	// Create non blocking socket
	// Allows setting Deadlines on connections
	// And allows proper closing on other threads
	err = syscall.SetNonblock(int(fd.Fd()), true)
	if err != nil {
		return conn, err
	}
	return conn, nil

}

func (vrrpNet vrrpNet) PollConn(ipConn *net.IPConn) {
	for {
		packet, err := vrrpNet.ReadPacket(ipConn)
		if errors.Is(err, net.ErrClosed) {
			break
		}
		if err != nil {
			klog.Errorf("%s", err.Error())
		}
		if packet != nil {
			VRRPBroker.packet <- packet
		}
	}
}

func (vrrpNet vrrpNet) ReadPacket(ipConn *net.IPConn) (*KowabungaVRRP, error) {
	buff := make([]byte, 1024)
	vrrpPacket := KowabungaVRRP{}
	err := ipConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return &vrrpPacket, err
	}
	_, err = ipConn.Read(buff)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, nil
		}
		return nil, err
	}
	if uint8(buff[8]) != KowabungaVRRPTTL {
		return &vrrpPacket, fmt.Errorf("TTL: %d should be 255 for vrrp", buff[8])
	}
	ipv4HeaderLen := int(buff[0]&0x0f) << 2
	err = vrrpPacket.DecodeFromBytes(buff[ipv4HeaderLen:])
	if err != nil {
		return &vrrpPacket, err
	}
	return &vrrpPacket, nil
}

func (vrrpNet vrrpNet) SendPacket(vrrpPacket *KowabungaVRRP, src *net.IP, dest []net.IP) error {
	for _, peer := range dest {
		bytes, err := vrrpPacket.Serialize(src, &peer, true)
		if err != nil {
			return err
		}
		if peer.IsMulticast() {
			for vrrpNet.multicastWriter == nil {
				time.Sleep(5 * time.Millisecond)
			}
			_, err = vrrpNet.multicastWriter.WriteTo(bytes, &net.IPAddr{IP: peer})
		} else {
			// TODO: better. Dirty hack, waiting to check conn has been init-ed
			for vrrpNet.unicastConn[src.String()] == nil {
				time.Sleep(5 * time.Millisecond)
			}
			_, err = vrrpNet.unicastConn[src.String()].conn.WriteTo(bytes, &net.IPAddr{IP: peer})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// advertisedIps: published IPs in the ARP packet
// ipsAssociatedMac: Corresponding mac for the published IPs (only 1 mac address supported)
// arpInterface: Interface used to send the ARP request
func (vrrpNet *vrrpNet) SendGratuitousARP(advertisedIps []net.IP, ipsAssociatedMac net.HardwareAddr, arpInterface *net.Interface) error {
	var err error

	if vrrpNet.arpClient == nil {
		vrrpNet.arpClient, err = arpConn(arpInterface)
		if err != nil {
			return err
		}
	}

	err = vrrpNet.arpClient.SetWriteDeadline(time.Now().Add(500 * time.Microsecond))
	if err != nil {
		return err
	}

	brAddr, err := net.ParseMAC(BroadcastAddress)
	if err != nil {
		return err
	}

	packet := &arp.Packet{
		HardwareType:       1,
		ProtocolType:       0x0800,
		HardwareAddrLength: 8,
		IPLength:           4,
		Operation:          2,
	}

	for _, ip := range advertisedIps {
		packet.SenderHardwareAddr = ipsAssociatedMac
		packet.SenderIP = netip.AddrFrom4([4]byte(ip.To4()))
		packet.TargetHardwareAddr = brAddr
		packet.TargetIP = netip.AddrFrom4([4]byte(ip.To4()))
		err = vrrpNet.arpClient.WriteTo(packet, brAddr)
		if err != nil {
			return err
		}
	}

	return nil
}

func arpConn(ifi *net.Interface) (*arp.Client, error) {
	arpClient, err := arp.Dial(ifi)
	if err != nil {
		return nil, err
	}
	return arpClient, nil
}
