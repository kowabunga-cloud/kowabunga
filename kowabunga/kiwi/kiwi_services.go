/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"fmt"
	"strings"

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
	Name      string
	Type      string
	Addresses []string
}

type KiwiReloadReply struct{}

func (k *Kiwi) Reload(args *KiwiReloadArgs, reply *KiwiReloadReply) error {
	klog.Infof("Reloading Kiwi agent configuration ...")

	klog.Debugf("Kiwi Config Args: %+v", args)

	dnsRecords := map[string]string{}
	for _, d := range args.Domains {
		for _, r := range d.Records {
			if r.Type == "A" {
				key := fmt.Sprintf("%s.%s.", r.Name, d)
				value := strings.Join(r.Addresses, ",")
				dnsRecords[key] = value
			}
		}
	}
	k.agent.dns.UpdateAllRecords(dnsRecords)

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
	// do nothing
	*reply = KiwiCreateDnsZoneReply{}
	return nil
}

/*
 * RPC DeleteDnsZone()
 */

type KiwiDeleteDnsZoneArgs struct {
	Domain string
}
type KiwiDeleteDnsZoneReply struct{}

func (k *Kiwi) DeleteDnsZone(args *KiwiDeleteDnsZoneArgs, reply *KiwiDeleteDnsZoneReply) error {
	// do nothing
	*reply = KiwiDeleteDnsZoneReply{}
	return nil
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
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	value := strings.Join(args.Addresses, ",")
	err := k.agent.dns.AddRecord(key, value)

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
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	value := strings.Join(args.Addresses, ",")
	err := k.agent.dns.UpdateRecord(key, value)

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
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	k.agent.dns.DeleteRecord(key)

	*reply = KiwiDeleteDnsRecordReply{}
	return nil
}
