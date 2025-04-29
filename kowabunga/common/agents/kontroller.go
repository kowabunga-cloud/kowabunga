/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

import (
	"fmt"

	"github.com/lorenzosaino/go-sysctl"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	KontrollerVersion  = "1.0"
	KontrollerAppNmame = "kowabunga-kontroller-agent"

	ErrorKontrollerNotRoot  = "Kontroller is not running with root privileges"
	ErrorKontrollerRegister = "Unable to register Kowabunga Kontroller Agent: %s"
	ErrorKontrollerSysctl   = "Unable to tune in sysctl setting '%s': %+v"
)

type KowabungaSysctlSetting struct {
	Key   string
	Value string
}

type KowabungaKontrollerAgent struct {
	*KowabungaAgent
	services map[string]*ManagedService
}

func NewKowabungaKontrollerAgent(services map[string]*ManagedService) (*KowabungaKontrollerAgent, error) {
	settings, err := common.GetCloudInitMetadataDataSettings()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	agent := &KowabungaKontrollerAgent{
		KowabungaAgent: NewKowabungaAgent(settings.ControllerAgentID, common.KowabungaControllerAgent, settings.ControllerEndpoint, settings.ControllerAgentToken),
		services:       services,
	}

	err = agent.RegisterServices(newKontroller(agent))

	return agent, err
}

func KontrollerDaemon(services map[string]*ManagedService, settings []KowabungaSysctlSetting) error {
	if !common.IsRoot() {
		klog.Error(ErrorKontrollerNotRoot)
		return fmt.Errorf("%s", ErrorKontrollerNotRoot)
	}

	ka, err := NewKowabungaKontrollerAgent(services)
	if err != nil {
		return fmt.Errorf(ErrorKontrollerRegister, err)
	}

	// tune in sysctl configuration, if any ...
	if len(settings) > 0 {
		klog.Infof("Tuning in Sysctl settings ...")
	}
	for _, sys := range settings {
		err := sysctl.Set(sys.Key, sys.Value)
		if err != nil {
			return fmt.Errorf(ErrorKontrollerSysctl, sys.Key, err)
		}
		klog.Debugf("Sysctl: set %s to %s", sys.Key, sys.Value)
	}

	ka.Run()

	return nil
}
