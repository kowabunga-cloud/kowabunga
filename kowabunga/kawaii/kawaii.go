/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kawaii

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents/templates"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
)

var kawaiiServices = map[string]*agents.ManagedService{
	"nftables": {
		BinaryPath: "", //TODO: Later use for binary upgrade mgmt
		UnitName:   "nftables.service",
		User:       "root",
		Group:      "root",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.NftablesFirewallGoTmpl,
				TargetPath:      "/etc/nft-network/firewall.nft",
			},
			{
				TemplateContent: templates.NftablesNatsGoTmpl,
				TargetPath:      "/etc/nft-network/nats.nft",
			},
			{
				TemplateContent: templates.NftablesConfGoTmpl,
				TargetPath:      "/etc/nftables.conf",
			},
		},
	},
	"keepalived": {
		BinaryPath: "",
		UnitName:   "keepalived.service",
		User:       "root",
		Group:      "root",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.KeepalivedConfTemplate("kawaii"),
				TargetPath:      "/etc/keepalived/keepalived.conf",
			},
			{
				TemplateContent: templates.KeepalivedNotifyGoTmpl,
				TargetPath:      "/etc/keepalived/notify.sh",
				IsExecutable:    true,
			},
		},
	},
	"strongswan": {
		BinaryPath: "", //TODO: Later use for binary upgrade mgmt
		UnitName:   "strongswan.service",
		User:       "root",
		Group:      "root",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.IPsecSwanctlConfGoTmpl,
				TargetPath:      "/etc/swanctl/swanctl.conf",
			},
			{
				TemplateContent: templates.IPsecCharonLoggingGoTmpl,
				TargetPath:      "/etc/strongswan.d/charon-logging.conf",
			},
			{
				TemplateContent: templates.IPsecCharonGoTmpl,
				TargetPath:      "/etc/strongswan.d/charon.conf",
			},
			{
				TemplateContent: templates.IPsecCharonLogrotateGoTmpl,
				TargetPath:      "/etc/logrotate.d/charon",
			},
		},
		Pre: []func(metadata *metadata.InstanceMetadata, args ...any) error{
			SetXFRMInterfaces,
			RemoveXFRMInterfaces,
		},
		Reload: []func(metadata *metadata.InstanceMetadata, args ...any) error{swanctlReload},
	},
	"peeringKontroller": {
		User:  "root",
		Group: "root",
		Pre:   []func(metadata *metadata.InstanceMetadata, args ...any) error{},
	},
}

var kawaiiSysctlSettings = []agents.KowabungaSysctlSetting{
	{
		Key:   "net.ipv4.ip_forward",
		Value: "1",
	},
	{
		Key:   "net.netfilter.nf_conntrack_max",
		Value: "524288",
	},
	// Strongswan
	{
		Key:   "net.ipv4.conf.all.accept_redirects",
		Value: "0",
	},
}

func swanctlReload(metadata *metadata.InstanceMetadata, args ...any) error {
	path, err := common.LookupBinary("swanctl")
	if err != nil {
		return err
	}
	err = common.BinExec(path, "", []string{"--load-all"}, []string{})
	if err != nil {
		return err
	}
	return nil
}

func Daemonize() error {
	return agents.KontrollerDaemon(kawaiiServices, kawaiiSysctlSettings)
}
