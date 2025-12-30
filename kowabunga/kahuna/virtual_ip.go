/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/common/metadata"
)

const (
	VrrpPriorityMaster = 150
	VrrpPriorityBackup = 100
)

type VirtualIP struct {
	VRRP        int    `bson:"vrrp_id"`
	Interface   string `bson:"interface"`
	VIP         string `bson:"vip"`
	Priority    int    `bson:"priority"`
	NetMaskSize int    `bson:"mask"`
	Public      bool   `bson:"public"`
}

func (ip *VirtualIP) Metadata() metadata.VirtualIpMetadata {
	return metadata.VirtualIpMetadata{
		VRRP:        ip.VRRP,
		Interface:   ip.Interface,
		VIP:         ip.VIP,
		Priority:    ip.Priority,
		NetMaskSize: ip.NetMaskSize,
		Public:      ip.Public,
	}
}
