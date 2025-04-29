/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/vishvananda/netlink"
)

const (
	ErrorKowarpNotRoot = "Kowarp is not running with root privileges"
)

var VirtualRouters map[uint8]*virtualRouter
var VRRPBroker *RxBroker = &RxBroker{
	registerVirtualRouters:   make(chan *virtualRouter),
	deregisterVirtualRouters: make(chan *virtualRouter),
	packet:                   make(chan *KowabungaVRRP),
	event:                    make(chan EVENT),
}

type Kowarp struct{}

func (k *Kowarp) Run() {
	if !common.IsRoot() {
		klog.Fatal(ErrorKowarpNotRoot)
		return
	}

	// System Interrupts channel
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	VirtualRouters = make(map[uint8]*virtualRouter)

	// Prepare our channel buffer for panic or unhandled errors
	routinesErrorsFeedback := make(chan error)

	// Start internal broker
	klog.Infof("Starting RxBroker")
	go VRRPBroker.Run(routinesErrorsFeedback)

	//Init loading
	k.LoadAndConfigureRouters(routinesErrorsFeedback)
	configTimer := *time.NewTicker(time.Duration(60) * time.Second)
	for {
		select {
		case <-configTimer.C:
			k.LoadAndConfigureRouters(routinesErrorsFeedback)
		case routineUnhandledError := <-routinesErrorsFeedback:
			klog.Errorf("%s", routineUnhandledError.Error())
		case sig := <-sigChannel:
			klog.Infof("Received Signal : %s", sig.String())
			if sig == os.Interrupt || sig == os.Kill {
				for _, vr := range VirtualRouters {
					klog.Infof("Shutting down VR ID %d", vr.ID)
					vr.Stop()
				}
			}
			klog.Infof("Shutting down Network broker")
			VRRPBroker.Stop()
			//Letting time for goroutines to close properly
			time.Sleep(200 * time.Millisecond)
			os.Exit(0)
		}
	}

}

func (k *Kowarp) LoadAndConfigureRouters(routinesErrorsFeedback chan<- error) {
	loadedRouters, err := k.loadConfig()
	if err != nil {
		klog.Errorf("Could not load config : %s", err.Error())
	}

	for _, loadedRouter := range loadedRouters {
		if VirtualRouters[loadedRouter.ID] == nil {
			VirtualRouters[loadedRouter.ID] = loadedRouter
			klog.Infof("Starting VR ID %d", loadedRouter.ID)
			go VirtualRouters[loadedRouter.ID].Start(routinesErrorsFeedback)
		} else if VirtualRouters[loadedRouter.ID].HasDiff(loadedRouter) {
			klog.Infof("Virtual Router %d : Configuration was updated. Reloading...")
			VirtualRouters[loadedRouter.ID].Stop()
			VirtualRouters[loadedRouter.ID] = loadedRouter
			go VirtualRouters[loadedRouter.ID].Start(routinesErrorsFeedback)
		}
	}

	// Shutdown non configured routers
	for _, runningVr := range VirtualRouters {
		shutdown := true
		for _, loadedRouter := range loadedRouters {
			if runningVr.ID == loadedRouter.ID {
				shutdown = false
			}
		}
		if shutdown {
			klog.Infof("Shutting down VR ID %d", runningVr.ID)
			runningVr.Stop()
		}
	}
	klog.Infof("New routers loaded")
}

// Read Config
func (*Kowarp) loadConfig() ([]*virtualRouter, error) {
	klog.Infof("Reloading instance controller configuration ...")
	var routers []*virtualRouter
	settings, err := common.GetCloudInitMetadataDataSettings()
	if err != nil {
		return nil, err
	}

	meta, err := common.GetInstanceMetadata(settings)
	if err != nil {
		return nil, err
	}
	controlInterface, err := net.InterfaceByName(meta.Kawaii.VrrpControlInterface)
	if err != nil {
		return routers, err
	}

	for _, vip := range meta.Kawaii.VirtualIPs {

		vrrpId := vip.VRRP
		peers := meta.Peers
		var peersIPs []net.IP
		for _, peer := range peers {
			peersIPs = append(peersIPs, net.ParseIP(peer))
		}
		priority := vip.Priority
		ip := vip.VIP
		mask := vip.NetMaskSize
		vipStr := fmt.Sprintf("%s/%d", ip, mask)

		virtualIP, err := netlink.ParseAddr(vipStr)
		if err != nil {
			klog.Errorf("Could not parse VIP from config")
		}
		virtualIPs := []netlink.Addr{*virtualIP}

		VIPinterface := vip.Interface

		if vip.Public {
			defaultDest := net.IPNet{
				IP:   net.IP{0, 0, 0, 0},
				Mask: net.IPMask{0, 0, 0, 0},
			}

			publicGateway := net.ParseIP(meta.Kawaii.PublicGateway)
			publicInterface, err := net.InterfaceByName(meta.Kawaii.PublicInterface)
			if err != nil {
				return routers, err
			}

			routes := []VirtualRoute{{
				Destination: defaultDest,
				Gateway:     publicGateway,
				Interface:   *publicInterface,
			},
			}
			vr, err := NewVirtualRouter(
				uint8(vrrpId),
				peersIPs,
				uint8(priority),
				virtualIPs,
				100,
				controlInterface.Name,
				VIPinterface,
				nil,
				false,
				routes,
			)
			if err != nil {
				return routers, err
			}
			routers = append(routers, vr)
		}
	}
	return routers, nil
}
