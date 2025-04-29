/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

type Kontroller struct {
	agent *KowabungaKontrollerAgent
}

func newKontroller(agent *KowabungaKontrollerAgent) *Kontroller {
	return &Kontroller{
		agent: agent,
	}
}

/*
 * RPC Capabilities()
 */

func (k *Kontroller) Capabilities(args *CapabilitiesArgs, reply *CapabilitiesReply) error {
	klog.Infof("Querying for RPC capabilities ...")
	*reply = CapabilitiesReply{
		Version: KontrollerVersion,
		Methods: k.agent.rpcServer.GetServices(),
	}
	return nil
}

/*
 * RPC Reload()
 */

type KontrollerReloadArgs struct{}
type KontrollerReloadReply struct{}

func (k *Kontroller) Reload(args *KontrollerReloadArgs, reply *KontrollerReloadReply) error {
	klog.Infof("Reloading instance controller configuration ...")

	settings, err := common.GetCloudInitMetadataDataSettings()
	if err != nil {
		return err
	}

	meta, err := common.GetInstanceMetadataRaw(settings)
	if err != nil {
		return err
	}
	metadataStruct, err := common.MetadataRawToStruct(meta)
	if err != nil {
		return err
	}

	for _, svc := range k.agent.services {
		var err error
		svcRestartRequired := false

		for _, preFunction := range svc.Pre {
			err = preFunction(metadataStruct, nil)
			if err != nil {
				klog.Errorf("Failed running preFlight function : %s", err.Error())
			}
		}

		updated, err := svc.TemplateConfigs(meta)
		if err != nil {
			klog.Errorf("Error templating config for %s : %s", svc.UnitName, err)
			return err
		}

		svcRestartRequired = updated

		if !updated {
			// If stopped, try restarting it. A svc have no reason to be stopped
			svcStarted, err := svc.IsServiceStarted()
			if err != nil {
				klog.Errorf("Could not check %s service status", svc.UnitName)
				return err
			}
			svcRestartRequired = !svcStarted
		}

		if svcRestartRequired {
			err := svc.ReloadOrRestart(metadataStruct)
			if err != nil {
				klog.Errorf("%s", err.Error())
				return err
			}

			// Post
			for _, postFunction := range svc.Post {
				err = postFunction(metadataStruct, nil)
				if err != nil {
					klog.Errorf("Failed running post reload function : %s", err.Error())
				}
			}
		}
	}

	*reply = KontrollerReloadReply{}
	return nil
}
