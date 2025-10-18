/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

type Kiwi struct {
	agent *KiwiAgent
}

func newKiwi(agent *KiwiAgent) *Kiwi {
	return &Kiwi{
		agent: agent,
	}
}

/*
 * RPC Capabilities()
 */

func (k *Kiwi) Capabilities(args *agents.CapabilitiesArgs, reply *agents.CapabilitiesReply) error {
	*reply = agents.CapabilitiesReply{
		Version: KiwiVersion,
		Methods: k.agent.RpcServer().GetServices(),
	}
	return nil
}

/*
 * RPC Reload()
 */

type KiwiReloadArgs struct {
	Domains []KiwiReloadArgsDomain
}

type KiwiReloadArgsDomain struct {
	Name    string
	Records []KiwiReloadArgsRecord
}

type KiwiReloadArgsRecord struct {
	Type      string
	Addresses []string
}

type KiwiReloadReply struct{}

func (k *Kiwi) Reload(args *KiwiReloadArgs, reply *KiwiReloadReply) error {
	klog.Infof("Reloading Kiwi agent configuration ...")

	klog.Debugf("Kiwi Config Args: %+v", args)

	*reply = KiwiReloadReply{}
	return nil
}

/*
 * RPC CreateDnsZone()
 */

type KiwiCreateDnsZoneArgs struct {
	Domain string
}
type KiwiCreateDnsZoneReply struct{}

func (k *Kiwi) CreateDnsZone(args *KiwiCreateDnsZoneArgs, reply *KiwiCreateDnsZoneReply) error {
	err := k.agent.pcs.CreateDnsZone(args.Domain)
	*reply = KiwiCreateDnsZoneReply{}
	return err
}

/*
 * RPC DeleteDnsZone()
 */

type KiwiDeleteDnsZoneArgs struct {
	Domain string
}
type KiwiDeleteDnsZoneReply struct{}

func (k *Kiwi) DeleteDnsZone(args *KiwiDeleteDnsZoneArgs, reply *KiwiDeleteDnsZoneReply) error {
	err := k.agent.pcs.DeleteDnsZone(args.Domain)
	*reply = KiwiDeleteDnsZoneReply{}
	return err
}

/*
 * RPC CreateDnsRecord()
 */

type KiwiCreateDnsRecordArgs struct {
	Domain    string
	Entry     string
	Addresses []string
}
type KiwiCreateDnsRecordReply struct{}

func (k *Kiwi) CreateDnsRecord(args *KiwiCreateDnsRecordArgs, reply *KiwiCreateDnsRecordReply) error {
	err := k.agent.pcs.CreateDnsRecord(args.Domain, args.Entry, args.Addresses)
	*reply = KiwiCreateDnsRecordReply{}
	return err
}

/*
 * RPC UpdateDnsRecord()
 */

type KiwiUpdateDnsRecordArgs struct {
	Domain    string
	Entry     string
	Addresses []string
}
type KiwiUpdateDnsRecordReply struct{}

func (k *Kiwi) UpdateDnsRecord(args *KiwiUpdateDnsRecordArgs, reply *KiwiUpdateDnsRecordReply) error {
	err := k.agent.pcs.UpdateDnsRecord(args.Domain, args.Entry, args.Addresses)
	*reply = KiwiUpdateDnsRecordReply{}
	return err
}

/*
 * RPC DeleteDnsRecord()
 */

type KiwiDeleteDnsRecordArgs struct {
	Domain string
	Entry  string
}
type KiwiDeleteDnsRecordReply struct{}

func (k *Kiwi) DeleteDnsRecord(args *KiwiDeleteDnsRecordArgs, reply *KiwiDeleteDnsRecordReply) error {
	err := k.agent.pcs.DeleteDnsRecord(args.Domain, args.Entry)
	*reply = KiwiDeleteDnsRecordReply{}
	return err
}
