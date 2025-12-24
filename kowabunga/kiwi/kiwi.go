/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	KiwiVersion  = "1.1"
	KiwiAppNmame = "kowabunga-kiwi"
)

type KiwiAgent struct {
	*agents.KowabungaAgent
	dns *DnsServer
}

func (k *KiwiAgent) Shutdown() {
	err := k.dns.Stop()
	if err != nil {
		klog.Error(err)
	}
}

func NewKiwiAgent(cfg *KiwiAgentConfig) (*KiwiAgent, error) {
	agent := &KiwiAgent{
		KowabungaAgent: agents.NewKowabungaAgent(cfg.Global.ID, common.KowabungaKiwiAgent, cfg.Global.Endpoint, cfg.Global.APIKey),
		dns:            nil,
	}
	agent.PostFlight = agent.Shutdown

	dns, err := NewDnsServer(cfg.DNS)
	if err != nil {
		klog.Error(err)
		return agent, err
	}
	agent.dns = dns

	err = dns.Start()
	if err != nil {
		klog.Error(err)
		return agent, err
	}

	err = agent.RegisterServices(newKiwi(agent))
	return agent, err
}

func Daemonize() error {
	// parsing commands
	cfgFile, debug := agents.ParseCommands()

	cfg, err := KiwiConfigParser(cfgFile)
	if err != nil {
		return fmt.Errorf("config: unable to unmarshal config (%s)", err)
	}

	// init our logger
	logLevel := cfg.Global.LogLevel
	if debug {
		logLevel = "DEBUG"
	}
	klog.Init(KiwiAppNmame, []klog.LoggerConfiguration{
		{
			Type:    "console",
			Enabled: true,
			Level:   logLevel,
		},
	})

	ka, err := NewKiwiAgent(cfg)
	if err != nil {
		return fmt.Errorf("unable to register Kowabunga Kiwi agent: %s", err)
	}

	ka.Run()

	return nil
}
