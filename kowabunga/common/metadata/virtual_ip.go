/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package metadata

type VirtualIpMetadata struct {
	VRRP        int    `json:"vrrp_id"`
	Interface   string `json:"interface"`
	VIP         string `json:"vip"`
	Priority    int    `json:"priority"`
	NetMaskSize int    `json:"mask"`
	Public      bool   `json:"public"`
}
