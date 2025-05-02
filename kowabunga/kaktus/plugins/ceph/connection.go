/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"time"

	"github.com/ceph/go-ceph/rados"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	CephCliBinary = "ceph"

	CephConnectionTimeout = 5 * time.Second
	CephPingTimeout       = 30 * time.Second

	CephMonitorDefaultPort = 3300

	CephNoConnectionError = "no associated ceph monitor connection"
)

func (ceph *ccs) Setup(name, address string, port int) error {

	if address == "" {
		return fmt.Errorf("invalid ceph configuration, missing address")
	}

	ceph.Name = name
	ceph.Address = address
	ceph.Port = port

	// check for supported protocol and ports
	if ceph.Port == 0 {
		ceph.Port = CephMonitorDefaultPort
	}

	// check for valid CLI binary
	bin, err := common.LookupBinary(CephCliBinary)
	if err != nil {
		return err
	}
	ceph.Bin = bin

	err = ceph.Connect()
	if err != nil {
		return err
	}

	host := fmt.Sprintf("%s:%d", ceph.Address, ceph.Port)
	klog.Infof("Successfully initiated Ceph monitor connection to %s", host)

	// maintain connection
	ceph.keepRunning = true
	klog.Infof("Register Ceph connection monitor for %s", host)
	go ceph.registerConnectionMonitor()

	return err
}

func (ceph *ccs) Connect() error {
	// ensure we're not already connected
	if ceph.Conn != nil {
		return nil
	}

	conn, err := rados.NewConn()
	if err != nil {
		return fmt.Errorf("failed to establish Rados cluster connection: %v", err)
	}

	err = conn.ReadDefaultConfigFile()
	if err != nil {
		return fmt.Errorf("unable to read Ceph default configuration file: %v", err)
	}

	err = conn.Connect()
	if err != nil {
		return fmt.Errorf("unable to connect to Ceph Rados cluster: %v", err)
	}

	ceph.Conn = conn

	return nil
}

func (ceph *ccs) Disconnect() {
	klog.Infof("Disconnecting from Ceph ...")
	ceph.keepRunning = false
	ceph.Conn.Shutdown()
}

func (ceph *ccs) registerConnectionMonitor() {
	host := fmt.Sprintf("%s:%d", ceph.Address, ceph.Port)
	for ceph.keepRunning {
		_, err := ceph.Conn.PingMonitor(ceph.Name)
		if err != nil {
			klog.Warningf("ceph disconnection from %s has been detected", host)
			ceph.Disconnect()
			ceph.Conn = nil

			for {
				klog.Infof("Trying to reconnect to %s", host)
				err := ceph.Connect()
				if err != nil {
					klog.Error(err)
					time.Sleep(CephConnectionTimeout)
					continue
				}
				klog.Infof("Successfully reconnected to Ceph on %s", host)
				break
			}
		}

		time.Sleep(CephPingTimeout)
	}
}
