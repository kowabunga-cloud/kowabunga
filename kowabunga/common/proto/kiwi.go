/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package proto

const (
	RpcKiwiReload          = "Reload"
	RpcKiwiCreateDnsZone   = "CreateDnsZone"
	RpcKiwiDeleteDnsZone   = "DeleteDnsZone"
	RpcKiwiCreateDnsRecord = "CreateDnsRecord"
	RpcKiwiUpdateDnsRecord = "UpdateDnsRecord"
	RpcKiwiDeleteDnsRecord = "DeleteDnsRecord"
)

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

/*
 * RPC CreateDnsZone()
 */

type KiwiCreateDnsZoneArgs struct {
	Domain string
}

type KiwiCreateDnsZoneReply struct{}

/*
 * RPC DeleteDnsZone()
 */

type KiwiDeleteDnsZoneArgs struct {
	Domain string
}

type KiwiDeleteDnsZoneReply struct{}

/*
 * RPC CreateDnsRecord()
 */

type KiwiCreateDnsRecordArgs struct {
	Domain    string
	Entry     string
	Addresses []string
}

type KiwiCreateDnsRecordReply struct{}

/*
 * RPC UpdateDnsRecord()
 */

type KiwiUpdateDnsRecordArgs struct {
	Domain    string
	Entry     string
	Addresses []string
}

type KiwiUpdateDnsRecordReply struct{}

/*
 * RPC DeleteDnsRecord()
 */

type KiwiDeleteDnsRecordArgs struct {
	Domain string
	Entry  string
}

type KiwiDeleteDnsRecordReply struct{}
