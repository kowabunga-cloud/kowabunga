/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"fmt"
	"net"
	"time"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/vishvananda/netlink"
)

const (
	KowabungaVRRPDefaultAdvertisementInterval uint16 = 100 //in centisecond
	KowabungaVRRPDefaultRouteMetric           int    = 50
)

var KowabungaVRRPVMacPrefix = "00-00-5E-00-01-"

type EVENT byte

const (
	SHUTDOWN EVENT = iota
	START
	DOWN
)

type STATE byte

const (
	INIT STATE = iota
	MASTER
	BACKUP
)

type VirtualRoute struct {
	Destination net.IPNet
	Gateway     net.IP
	Interface   net.Interface
	Metric      int
}

func (r1 *VirtualRoute) Equal(r2 *VirtualRoute) bool {
	if r1.Destination.IP.Equal(r2.Destination.IP) &&
		r1.Gateway.Equal(r2.Gateway) &&
		r1.Interface.Name == r2.Interface.Name &&
		r1.Metric == r2.Metric {
		return true
	}
	return false
}

type virtualRouter struct {
	ID                   uint8
	Priority             uint8
	ProtectedIPAddresses []netlink.Addr
	state                STATE

	AdvertisementInterval  uint16
	MasterAdverInterval    uint16
	MasterDownInterval     uint16
	skewTime               uint16
	PreemptMode            bool // Always true
	AcceptMode             bool // Always False
	AdvertisementInterface *net.Interface
	// RFC extras
	Peers             []net.IP       // If set, unicast mode will be used
	Interface         *net.Interface // Must Match an existing interface if vmac set to false
	PreferredSourceIP net.IP         // If not set, defaults to Interface first IP
	UseVmac           bool           // Not supported yet
	Routes            []VirtualRoute

	// Event channels

	eventChannel chan EVENT
	packet       chan *KowabungaVRRP

	// Timers
	adverTimer      time.Ticker
	masterDownTimer time.Ticker
}

func NewVirtualRouter(
	id uint8,
	peers []net.IP,
	priority uint8,
	ipAddrs []netlink.Addr,
	advertisementInterval uint16,
	advertisementInterfaceName string,
	vrrpInterfaceName string,
	preferredSourceIP net.IP,
	useVmac bool,
	virtualRoutes []VirtualRoute) (*virtualRouter, error) {

	if advertisementInterval == 0 || advertisementInterval > 4095 {
		klog.Infof("virtual router config: Advertisement Interval not set or >4095 for router ID %d. Setting default : %d", id, KowabungaVRRPDefaultAdvertisementInterval)
		advertisementInterval = KowabungaVRRPDefaultAdvertisementInterval
	}
	var advItf *net.Interface
	var vrrpItf *net.Interface
	var err error
	if advertisementInterfaceName != "" {
		advItf, err = net.InterfaceByName(advertisementInterfaceName)
	} else {
		advItf, err = findFirstPrivateInterface()
	}
	if err != nil {
		return nil, err
	}

	if vrrpInterfaceName != "" {
		vrrpItf, err = net.InterfaceByName(vrrpInterfaceName)
	} else {
		vrrpItf, err = findFirstPrivateInterface()
	}
	if err != nil {
		return nil, err
	}

	if preferredSourceIP == nil {
		preferredSourceIP, err = findIPbyInterface(advItf)
		if err != nil {
			return nil, err
		}
	}
	skewTime := (256 - uint16(priority)) * advertisementInterval / 256
	eventChannel := make(chan EVENT, 1)
	packetChannel := make(chan *KowabungaVRRP)

	vr := &virtualRouter{
		ID:                     id,
		state:                  INIT,
		Peers:                  peers,
		Priority:               priority,
		ProtectedIPAddresses:   ipAddrs,
		AdvertisementInterval:  advertisementInterval,
		skewTime:               skewTime,
		MasterDownInterval:     3*advertisementInterval + skewTime,
		PreemptMode:            true,
		AcceptMode:             false,
		AdvertisementInterface: advItf,
		Interface:              vrrpItf,
		PreferredSourceIP:      preferredSourceIP,
		UseVmac:                useVmac,
		Routes:                 virtualRoutes,
		eventChannel:           eventChannel,
		packet:                 packetChannel,
	}
	return vr, nil
}

func (vr *virtualRouter) Initialize() error {
	klog.Infof("Virtual router %d entering INIT mode", vr.ID)
	if vr.Priority == 0xFF {
		err := vr.sendAdvertisements()
		if err != nil {
			klog.Errorf("Could not send Advertisement for vr ID %d : %s", vr.ID, err.Error())
		}
		klog.Debugf("Advertisement sent for router id %d", vr.ID)
		err = vr.sendGratuitousARP()
		if err != nil {
			klog.Errorf("Could not send gratuitous ARP for vr ID %d : %s", vr.ID, err.Error())
		}
		vr.adverTimer = *time.NewTicker(time.Duration(vr.AdvertisementInterval*10) * time.Millisecond)
		vr.state = MASTER
	} else {
		vr.MasterAdverInterval = vr.AdvertisementInterval
		vr.skewTime = (256 - uint16(vr.Priority)) * vr.MasterAdverInterval / 256
		vr.masterDownTimer = *time.NewTicker(time.Duration(vr.MasterDownInterval*10) * time.Millisecond)
		vr.state = BACKUP
	}
	return nil
}

func (vr *virtualRouter) MasterMode() {
	klog.Infof("Virtual router %d entering MASTER mode", vr.ID)
	if !vr.UseVmac {
		vr.UpdateNet()
	}
	for {
		select {
		case event := <-vr.eventChannel:
			if event == SHUTDOWN {
				priority := vr.Priority
				vr.Priority = 0
				err := vr.sendAdvertisements()
				if err != nil {
					klog.Errorf("VR %d shutdown : Could not send advertisement", vr.ID)
				}
				vr.Priority = priority
				vr.state = INIT
				return
			}
		case <-vr.adverTimer.C:
			err := vr.sendAdvertisements()
			if err != nil {
				klog.Errorf("VR %d: Could not send advertisement", vr.ID)
			}
		case packet := <-vr.packet:
			klog.Debugf("Virtual router %d : received vrrp packet", vr.ID)
			if packet.Priority == 0 {
				err := vr.sendAdvertisements()
				if err != nil {
					klog.Debugf("Master : failed to send advertisement.")
				}
			} else if (vr.Priority < packet.Priority) || (packet.Priority == vr.Priority && ipLargerThan(packet.pseudoHeader.SourceAddress, vr.PreferredSourceIP)) {
				klog.Debugf("Virtual Router %d : received VRRP packet with higher priority", vr.ID)
				if !vr.UseVmac {
					vr.CleanNet()
				}
				vr.MasterAdverInterval = packet.AdverInt
				vr.skewTime = (256 - uint16(vr.Priority)) * vr.MasterAdverInterval / 256
				vr.MasterDownInterval = 3*vr.MasterAdverInterval + vr.skewTime
				vr.masterDownTimer = *time.NewTicker(time.Duration(vr.MasterDownInterval*10) * time.Millisecond)
				vr.state = BACKUP
				return
			}
		}
	}
}

func (vr *virtualRouter) BackupMode() {
	klog.Infof("Virtual router %d entering BACKUP mode", vr.ID)
	var interval uint16
	for {
		select {
		case event := <-vr.eventChannel:
			if event == SHUTDOWN {
				klog.Infof("VR %d : Shutdown received", vr.ID)
				vr.state = INIT
				return
			}
		case <-vr.masterDownTimer.C:
			err := vr.sendAdvertisements()
			if err != nil {
				klog.Errorf("Master Down : failed to send advertisement")
			}
			err = vr.sendGratuitousARP()
			if err != nil {
				klog.Errorf("Master Down : failed to gratuitous ARP")
			}
			vr.adverTimer = *time.NewTicker(time.Duration(vr.AdvertisementInterval*10) * time.Millisecond)
			vr.state = MASTER
			return
		case packet := <-vr.packet:
			if packet.Priority == 0 {
				interval = vr.AdvertisementInterval
			} else if !vr.PreemptMode || packet.Priority >= vr.Priority {
				vr.MasterAdverInterval = packet.AdverInt
				vr.MasterDownInterval = 3*vr.MasterAdverInterval + vr.skewTime
				interval = vr.MasterDownInterval
			}
		}
		vr.masterDownTimer = *time.NewTicker(time.Duration(interval*10) * time.Millisecond)
	}
}

func (vr *virtualRouter) Start(routineUnhandledError chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				routineUnhandledError <- fmt.Errorf("virtual Router failed %d : %w", vr.ID, err)
			} else {
				routineUnhandledError <- fmt.Errorf("virtual Router %d : Panic happened with %v", vr.ID, r)
			}
		}
	}()

	VRRPBroker.registerVirtualRouters <- vr
	vr.eventChannel <- START
	for {
		switch vr.state {
		case INIT:
			if len(vr.eventChannel) > 0 {
				event := <-vr.eventChannel
				if event == START {
					klog.Infof("Started VR ID %d", vr.ID)
					err := vr.Initialize()
					if err != nil {
						klog.Errorf("Could not Initialize virtual router %d :", vr.ID)
						klog.Infof(err.Error())
					}
				}
			} else {
				return
			}
		case MASTER:
			vr.MasterMode()
		case BACKUP:
			vr.BackupMode()
		}
	}
}

func (vr *virtualRouter) Stop() {
	VRRPBroker.deregisterVirtualRouters <- vr
	vr.eventChannel <- SHUTDOWN
	vr.CleanNet()
}

func (vr *virtualRouter) sendAdvertisements() error {
	var err error
	var advertisedIps []net.IP
	for _, ip := range vr.ProtectedIPAddresses {
		advertisedIps = append(advertisedIps, ip.IP)
	}
	vrrpPacket := KowabungaVRRP{
		Version:      KowabungaVRRPVersion,
		Type:         KowabungaVRRPAdvertisement,
		VirtualRtrID: vr.ID,
		Priority:     vr.Priority,
		CountIPAddr:  uint8(len(vr.ProtectedIPAddresses)),
		AdverInt:     vr.AdvertisementInterval,
		IPAddresses:  advertisedIps,
	}
	klog.Debugf("Sending advertisement for router %d", vr.ID)
	if len(vr.Peers) == 0 {
		multiAddr := []net.IP{
			net.ParseIP(KowabungaVRRPMulticastAddress),
		}
		err = VRRPNetworkBroker.SendPacket(&vrrpPacket, &vr.PreferredSourceIP, multiAddr)
	} else {
		err = VRRPNetworkBroker.SendPacket(&vrrpPacket, &vr.PreferredSourceIP, vr.Peers)
	}
	return err
}

func (vr *virtualRouter) sendGratuitousARP() error {
	var hwAddr net.HardwareAddr
	var err error
	var advertisedIps []net.IP
	for _, ip := range vr.ProtectedIPAddresses {
		advertisedIps = append(advertisedIps, ip.IP)
	}
	if vr.UseVmac {
		hwAddr, err = net.ParseMAC(fmt.Sprintf("%s%x", KowabungaVRRPVMacPrefix, vr.ID))
		if err != nil {
			return err
		}
	} else {
		hwAddr = vr.Interface.HardwareAddr
	}
	klog.Debugf("Sending gratuitous ARP for router %d", vr.ID)
	err = VRRPNetworkBroker.SendGratuitousARP(advertisedIps, hwAddr, vr.AdvertisementInterface)
	if err != nil {
		return err
	}
	return nil
}

func (vr *virtualRouter) UpdateNet() {
	var err error
	for _, ip := range vr.ProtectedIPAddresses {
		err = addIPToInterface(vr.Interface, &ip)
		if err != nil {
			klog.Errorf("addIPToInterface: VR : %d; Could not add IP to interface. %s", vr.ID, err.Error())
		}
	}
	for _, route := range vr.Routes {
		var metric int
		if route.Metric != 0 {
			metric = route.Metric
		} else {
			metric = KowabungaVRRPDefaultRouteMetric
		}
		err = addRoute(&route.Interface, &route.Gateway, &route.Destination, metric)
		if err != nil {
			klog.Errorf("addRoute: VR : %d; Could not add route %s via %s dev %s => %s", vr.ID, route.Destination.String(), route.Gateway.String(), route.Interface.Name, err.Error())
		}
	}
}

func (vr *virtualRouter) CleanNet() {
	var err error
	for _, route := range vr.Routes {
		var metric int
		if route.Metric != 0 {
			metric = route.Metric
		} else {
			metric = KowabungaVRRPDefaultRouteMetric
		}
		err = removeRoute(&route.Interface, &route.Gateway, &route.Destination, metric)
		if err != nil {
			klog.Errorf("removeRoute: VR : %d; Could not remove route %s via %s dev %s => %s", vr.ID, route.Destination.String(), route.Gateway.String(), route.Interface.Name, err.Error())
		}
	}
	for _, ip := range vr.ProtectedIPAddresses {
		err = removeIPFromInterface(vr.Interface, &ip)
		if err != nil {
			klog.Errorf("removeIPFromInterface: VR : %d; Could not remove route to interface. %s", vr.ID, err.Error())
		}
	}
}

func (running *virtualRouter) HasDiff(loaded *virtualRouter) bool {
	if running.ID != loaded.ID {
		klog.Errorf("HasDiff : compared Virtual Routers do not share the same VR ID")
		return true
	}

	if running.Interface != loaded.Interface ||
		running.AdvertisementInterface != loaded.AdvertisementInterface ||
		running.Priority != loaded.Priority ||
		running.UseVmac != loaded.UseVmac ||
		running.AcceptMode != loaded.AcceptMode ||
		running.PreemptMode != loaded.PreemptMode {
		return true
	}

	if !ipListsEqual(running.Peers, loaded.Peers) {
		return true
	}

	if !ipAddrListsEqual(running.ProtectedIPAddresses, loaded.ProtectedIPAddresses) {
		return true
	}

	if !ipVirtualRoutesEqual(running.Routes, loaded.Routes) {
		return true
	}
	return false
}
