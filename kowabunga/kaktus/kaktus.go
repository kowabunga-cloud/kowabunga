/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"fmt"
	"plugin"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/agents"
	"github.com/kowabunga-cloud/common/klog"
)

const (
	KaktusVersion              = "1.0"
	KaktusAppNmame             = "kowabunga-kaktus"
	KaktusPluginCephEntryPoint = "CephConnectionSettings"
)

// Ceph Plugin Interface
type CephConnectionSettings interface {
	// Connection
	Setup(name, address string, port int) error
	Connect() error
	Disconnect()

	// RBD
	ListRbdVolumes(poolName string) ([]string, error)
	GetPoolStats(poolName string) (uint64, uint64, uint64, error)
	GetRbdVolumeInfos(poolName, volName string) (uint64, error)
	CreateRbdVolume(poolName, volName string, size uint64) error
	CreateRbdVolumeFromBinData(poolName, volName string, size uint64, data []byte) error
	UpdateRbdVolumeFromBinData(poolName, volName string, size uint64, data []byte) error
	CreateRbdVolumeFromUrl(poolName, volName, url string) (uint64, error)
	CloneRbdVolume(poolName, srcName, dstName string, size uint64) error
	ResizeRbdVolume(poolName, volName string, size uint64) error
	DeleteRbdVolume(poolName, volName string, deleteSnapshots bool) error

	// CephFS
	ListVolumes() ([]string, error)
	ListSubVolumes(vol string) ([]string, error)
	CreateSubVolume(vol, sub string) (string, int64, error)
	DeleteSubVolume(vol, sub string) error
}

type KaktusAgent struct {
	*agents.KowabungaAgent
	lcs  *LibvirtConnectionSettings
	ceph CephConnectionSettings
	nfs  *NfsConnectionSettings
}

func (k *KaktusAgent) Shutdown() {
	err := k.lcs.Disconnect()
	if err != nil {
		klog.Error(err)
	}
	k.ceph.Disconnect()
}

func NewKaktusAgent(cfg *KaktusAgentConfig) (*KaktusAgent, error) {
	agent := &KaktusAgent{
		KowabungaAgent: agents.NewKowabungaAgent(cfg.Global.ID, common.KowabungaKaktusAgent, cfg.Global.Endpoint, cfg.Global.APIKey),
		lcs:            nil,
		ceph:           nil,
	}
	agent.PostFlight = agent.Shutdown

	lcs, err := NewLibvirtConnectionSettings(cfg)
	if err != nil {
		return agent, err
	}
	agent.lcs = lcs

	p, err := plugin.Open(cfg.Ceph.PluginLib)
	if err != nil {
		return agent, fmt.Errorf("unable to load Kaktus Ceph plugin: %v", err)
	}

	symCephConnectionSettings, err := p.Lookup(KaktusPluginCephEntryPoint)
	if err != nil {
		return agent, fmt.Errorf("unable to load Ceph plugin '%s' symbol: %v", KaktusPluginCephEntryPoint, err)
	}

	var ccs CephConnectionSettings
	ccs, ok := symCephConnectionSettings.(CephConnectionSettings)
	if !ok {
		return agent, err
	}
	agent.ceph = ccs

	err = agent.ceph.Setup(cfg.Ceph.Monitor.Name, cfg.Ceph.Monitor.Address, cfg.Ceph.Monitor.Port)
	if err != nil {
		return agent, err
	}

	nfs, err := NewNfsConnectionSettings()
	if err != nil {
		return agent, err
	}
	agent.nfs = nfs

	err = agent.RegisterServices(newKaktus(agent))
	return agent, err
}

func Daemonize() error {
	// parsing commands
	cfgFile, debug := agents.ParseCommands()

	cfg, err := KaktusConfigParser(cfgFile)
	if err != nil {
		return fmt.Errorf("config: unable to unmarshal config (%s)", err)
	}

	// init our logger
	logLevel := cfg.Global.LogLevel
	if debug {
		logLevel = "DEBUG"
	}
	klog.Init(KaktusAppNmame, []klog.LoggerConfiguration{
		{
			Type:    "console",
			Enabled: true,
			Level:   logLevel,
		},
	})

	ka, err := NewKaktusAgent(cfg)
	if err != nil {
		return fmt.Errorf("unable to register Kowabunga Kaktus agent: %s", err)
	}

	ka.Run()

	return nil
}
