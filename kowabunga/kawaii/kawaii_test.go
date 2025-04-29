/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kawaii

import (
	"fmt"
	"testing"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents/templates"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
	"github.com/vishvananda/netlink"
)

const (
	TestKawaiiServicesConfigDir = "/tmp/kowabunga/kawaii"
)

var testKawaiiServices = map[string]*agents.ManagedService{
	"nftables": {
		BinaryPath: "",
		UnitName:   "nftables",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.NftablesFirewallGoTmpl,
				TargetPath:      "firewall.nft",
			},
			{
				TemplateContent: templates.NftablesNatsGoTmpl,
				TargetPath:      "nats.nft",
			},
			{
				TemplateContent: templates.NftablesConfGoTmpl,
				TargetPath:      "nftables.conf",
			},
		},
	},
	"keepalived": {
		BinaryPath: "",
		UnitName:   "keepalived",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.KeepalivedConfTemplate("kawaii"),
				TargetPath:      "keepalived.conf",
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
				TargetPath:      "swanctl.conf",
			},
			{
				TemplateContent: templates.IPsecCharonLoggingGoTmpl,
				TargetPath:      "charon-logging.conf",
			},
			{
				TemplateContent: templates.IPsecCharonGoTmpl,
				TargetPath:      "charon.conf",
			},
			{
				TemplateContent: templates.IPsecCharonLogrotateGoTmpl,
				TargetPath:      "charon",
			},
		},
	},
}

var testKawaiiConfig = map[string]any{
	"kawaii": map[string]any{
		"ipsec_connections": []map[string]any{{
			"xfrm_id":                     "1",
			"name":                        "TESTIPSEC",
			"remote_peer":                 "97.8.9.10",
			"remote_subnet":               "10.4.0.0/24",
			"pre_shared_key":              "gibberish",
			"rekey":                       "240",
			"start_action":                "start",
			"dpd_action":                  "restart",
			"dpd_timeout":                 "240s",
			"phase1_lifetime":             "240s",
			"phase1_df_group":             "2",
			"phase1_integrity_algorithm":  "SHA1",
			"phase1_encryption_algorithm": "AES128",
			"phase2_lifetime":             "240s",
			"phase2_df_group":             "2",
			"phase2_integrity_algorithm":  "SHA1",
			"phase2_encryption_algorithm": "AES128",
			"ingress_rules": []map[string]any{{
				"protocol": "tcp",
				"ports":    "443",
				"action":   "allow",
			}}}},
		"public_interface":  "ens3",
		"private_interface": "ens4",
		"peering_interfaces": []string{
			"ens5",
		},
		"vrrp_control_interface": "ens4",
		"public_vip_addresses": []string{
			"60.0.0.1",
			"60.0.0.2",
		},
		"public_gw_address": "60.0.0.254",
		"virtual_ips": []map[string]any{
			{
				"vrrp_id":   1,
				"interface": "ens3",
				"vip":       "60.0.0.1",
				"priority":  150,
				"mask":      28,
				"public":    true,
			},
			{
				"vrrp_id":   2,
				"interface": "ens3",
				"vip":       "60.0.0.2",
				"priority":  150,
				"mask":      28,
				"public":    true,
			},
			{
				"vrrp_id":   2,
				"interface": "ens4",
				"vip":       "10.3.0.1",
				"priority":  150,
				"mask":      25,
				"public":    false,
			},
		},
		"fw_input_default":   "drop",
		"fw_output_default":  "accept",
		"fw_forward_default": "drop",
		"fw_input_extra_networks": []string{
			"10.5.0.0/22",
		},
		"fw_input_rules": []map[string]any{
			{
				"iifname":        "ens3",
				"oifname":        "ens4",
				"source_ip":      "0.0.0.0",
				"destination_ip": "0.0.0.0",
				"direction":      "out",
				"protocol":       "tcp",
				"ports":          "100-150",
				"action":         "forward",
			},
		},
		"fw_output_rules": []map[string]any{
			{
				"iifname":        "ens3",
				"oifname":        "ens4",
				"source_ip":      "0.0.0.0",
				"destination_ip": "0.0.0.0",
				"direction":      "out",
				"protocol":       "tcp",
				"ports":          "100-200",
				"action":         "forward",
			},
		},
		"fw_forward_rules": []map[string]any{
			{
				"iifname":        "ens3",
				"oifname":        "ens4",
				"source_ip":      "0.0.0.0",
				"destination_ip": "0.0.0.0",
				"direction":      "out",
				"protocol":       "tcp",
				"ports":          "100-300",
				"action":         "forward",
			},
		},
		"fw_nat_rules": []map[string]any{
			{
				"private_ip": "10.0.0.0",
				"public_ip":  "70.0.0.0",
				"protocol":   "tcp",
				"ports":      "100-200",
			},
		},
	},
}

func TestKawaiiTemplate(t *testing.T) {
	agents.AgentTestTemplate(t, testKawaiiServices, TestKawaiiServicesConfigDir, testKawaiiConfig)
}

func TestAddXfrmInt(t *testing.T) {
	xfrmAttr := netlink.NewLinkAttrs()
	xfrmAttr.Name = fmt.Sprintf("xfrm-%d", 1)
	itf := &netlink.Xfrmi{Ifid: 20, LinkAttrs: xfrmAttr}

	//itftest := &netlink.Bridge{LinkAttrs: xfrmAttr}
	handle, err := netlink.NewHandle()
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	err = handle.LinkAdd(itf)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestListXfrm(t *testing.T) {
	xfrmMetaNew := metadata.InstanceMetadata{
		Kawaii: &metadata.KawaiiMetadata{
			IPsecConnections: []metadata.KawaiiIPsecConnectionMetadata{
				{
					RemoteSubnet: "10.68.0.0/0",
					XfrmId:       1,
				},
				{
					RemoteSubnet: "10.69.0.0/0",
					XfrmId:       3,
				},
			},
		},
	}

	xfrmMetaRm := metadata.InstanceMetadata{
		Kawaii: &metadata.KawaiiMetadata{
			IPsecConnections: []metadata.KawaiiIPsecConnectionMetadata{},
		},
	}

	err := SetXFRMInterfaces(&xfrmMetaNew, nil)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	time.Sleep(5000)
	err = RemoveXFRMInterfaces(&xfrmMetaRm, nil)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
}
