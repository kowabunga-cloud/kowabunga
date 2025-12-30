/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kowarp

import (
	"fmt"
	"sync"

	"github.com/kowabunga-cloud/common/klog"
)

// Net => broker => appropriate vrID
// Virtual Router on Start must push its config to the Broker
// Brokers role is to manage reading on all sockets and forward
// packets to the appropriate virtualRouterId

var VRRPNetworkBroker vrrpNet = vrrpNet{
	unicastConn: make(map[string]*vrrpConn),
}

type RxBroker struct {
	registerVirtualRouters   chan *virtualRouter
	deregisterVirtualRouters chan *virtualRouter
	packet                   chan *KowabungaVRRP
	mu                       sync.Mutex
	event                    chan EVENT
}

func (rb *RxBroker) Run(routineUnhandledError chan error) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				routineUnhandledError <- fmt.Errorf("VRRPNetworkBroker failed : %w", err)
			} else {
				routineUnhandledError <- fmt.Errorf("VRRPNetworkBroker : Panic happened with %v", r)
			}
		}
	}()

	// Make sure channels are initialized
	if rb.registerVirtualRouters == nil {
		rb.registerVirtualRouters = make(chan *virtualRouter)
	}
	if rb.deregisterVirtualRouters == nil {
		rb.deregisterVirtualRouters = make(chan *virtualRouter)
	}
	if rb.packet == nil {
		rb.packet = make(chan *KowabungaVRRP)
	}
	if rb.event == nil {
		rb.event = make(chan EVENT)
	}

	for {
		select {
		case event := <-rb.event:
			if event == SHUTDOWN {
				return
			}
		case vr := <-rb.registerVirtualRouters:
			klog.Infof("RxBroker: Adding router id %d to configuration", vr.ID)
			rb.mu.Lock()
			err := VRRPNetworkBroker.addRouter(vr)
			if err != nil {
				klog.Errorf("Could not add router to list %s", err.Error())
			}
			rb.mu.Unlock()
		case vr := <-rb.deregisterVirtualRouters:
			klog.Infof("RxBroker: Removing router id %d to configuration", vr.ID)
			rb.mu.Lock()
			err := VRRPNetworkBroker.removeRouter(vr)
			if err != nil {
				klog.Errorf("Could not removed router from list %s", err.Error())
			}
			rb.mu.Unlock()
		case packet := <-rb.packet:
			if VirtualRouters[packet.VirtualRtrID] != nil {
				VirtualRouters[packet.VirtualRtrID].packet <- packet
			}
			// else the message is discarded without any message
		}
	}
}

func (rb *RxBroker) Stop() {
	rb.event <- SHUTDOWN
}
