/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"fmt"
	"strings"

	"github.com/kowabunga-cloud/common/agents"
	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/proto"
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

func (k *Kiwi) Reload(args *proto.KiwiReloadArgs, reply *proto.KiwiReloadReply) error {
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

	*reply = proto.KiwiReloadReply{}
	return nil
}

/*
 * RPC CreateDnsZone()
 */

func (k *Kiwi) CreateDnsZone(args *proto.KiwiCreateDnsZoneArgs, reply *proto.KiwiCreateDnsZoneReply) error {
	// do nothing
	*reply = proto.KiwiCreateDnsZoneReply{}
	return nil
}

/*
 * RPC DeleteDnsZone()
 */

func (k *Kiwi) DeleteDnsZone(args *proto.KiwiDeleteDnsZoneArgs, reply *proto.KiwiDeleteDnsZoneReply) error {
	// do nothing
	*reply = proto.KiwiDeleteDnsZoneReply{}
	return nil
}

/*
 * RPC CreateDnsRecord()
 */

func (k *Kiwi) CreateDnsRecord(args *proto.KiwiCreateDnsRecordArgs, reply *proto.KiwiCreateDnsRecordReply) error {
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	value := strings.Join(args.Addresses, ",")
	err := k.agent.dns.AddRecord(key, value)

	*reply = proto.KiwiCreateDnsRecordReply{}

	return err
}

/*
 * RPC UpdateDnsRecord()
 */

func (k *Kiwi) UpdateDnsRecord(args *proto.KiwiUpdateDnsRecordArgs, reply *proto.KiwiUpdateDnsRecordReply) error {
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	value := strings.Join(args.Addresses, ",")
	err := k.agent.dns.UpdateRecord(key, value)

	*reply = proto.KiwiUpdateDnsRecordReply{}
	return err
}

/*
 * RPC DeleteDnsRecord()
 */

func (k *Kiwi) DeleteDnsRecord(args *proto.KiwiDeleteDnsRecordArgs, reply *proto.KiwiDeleteDnsRecordReply) error {
	key := fmt.Sprintf("%s.%s.", args.Entry, args.Domain)
	k.agent.dns.DeleteRecord(key)

	*reply = proto.KiwiDeleteDnsRecordReply{}
	return nil
}
