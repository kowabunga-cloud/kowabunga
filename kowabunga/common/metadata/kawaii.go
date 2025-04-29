/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package metadata

type KawaiiMetadata struct {
	PublicInterface              string                          `json:"public_interface"`
	PrivateInterface             string                          `json:"private_interface"`
	PeeringInterfaces            []string                        `json:"peering_interfaces"`
	VrrpControlInterface         string                          `json:"vrrp_control_interface"`
	PublicVipAddresses           []string                        `json:"public_vip_addresses"`
	PublicGateway                string                          `json:"public_gw_address"`
	VirtualIPs                   []VirtualIpMetadata             `json:"virtual_ips"`
	FirewallDefaultInputPolicy   string                          `json:"fw_input_default"`
	FirewallDefaultOutputPolicy  string                          `json:"fw_output_default"`
	FirewallDefaultForwardPolicy string                          `json:"fw_forward_default"`
	FirewallInputExtraNetworks   []string                        `json:"fw_input_extra_networks"`
	FirewallInputRules           []KawaiiFirewallRuleMetadata    `json:"fw_input_rules"`
	FirewallOutputRules          []KawaiiFirewallRuleMetadata    `json:"fw_output_rules"`
	FirewallForwardRules         []KawaiiFirewallRuleMetadata    `json:"fw_forward_rules"`
	FirewallNatRules             []KawaiiFirewallNatRuleMetadata `json:"fw_nat_rules"`
	IPsecConnections             []KawaiiIPsecConnectionMetadata `json:"ipsec_connections"`
}

type KawaiiFirewallRuleMetadata struct {
	InputInterface  string `json:"iifname,omitempty"`
	OutputInterface string `json:"oifname,omitempty"`
	Source          string `json:"source_ip,omitempty"`
	Destination     string `json:"destination_ip,omitempty"`
	Direction       string `json:"direction,omitempty"`
	Protocol        string `json:"protocol"`
	Ports           string `json:"ports"`
	Action          string `json:"action"`
}

type KawaiiFirewallNatRuleMetadata struct {
	PrivateIP string `json:"private_ip"`
	PublicIP  string `json:"public_ip"`
	Protocol  string `json:"protocol"`
	Ports     string `json:"ports"`
}

type KawaiiIPsecConnectionMetadata struct {
	Name                      string                       `json:"name"`
	IP                        string                       `json:"ip"`
	XfrmId                    uint8                        `json:"xfrm_id"`
	RemotePeer                string                       `json:"remote_peer"`
	RemoteSubnet              string                       `json:"remote_subnet"`
	PreSharedKey              string                       `json:"pre_shared_key"`
	DpdTimeout                string                       `json:"dpd_timeout"`
	DpdTimeoutAction          string                       `json:"dpd_action"`
	StartAction               string                       `json:"start_action"`
	Rekey                     string                       `json:"rekey"`
	Phase1Lifetime            string                       `json:"phase1_lifetime"`
	Phase1DHGroup             string                       `json:"phase1_df_group"`
	Phase1IntegrityAlgorithm  string                       `json:"phase1_integrity_algorithm"`
	Phase1EncryptionAlgorithm string                       `json:"phase1_encryption_algorithm"`
	Phase2Lifetime            string                       `json:"phase2_lifetime"`
	Phase2DHGroup             string                       `json:"phase2_df_group"`
	Phase2IntegrityAlgorithm  string                       `json:"phase2_integrity_algorithm"`
	Phase2EncryptionAlgorithm string                       `json:"phase2_encryption_algorithm"`
	IngressRules              []KawaiiFirewallRuleMetadata `json:"ingress_rules"`
}
