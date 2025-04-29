/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package metadata

type InstanceMetadata struct {
	ID          string          `json:"instance-id"`
	Name        string          `json:"local-hostname"`
	Description string          `json:"description"`
	Kaktus      string          `json:"kaktus"`
	VCPUs       int64           `json:"vcpu"`
	Memory      int64           `json:"mem_gb"`
	LocalIPv4   string          `json:"local-ipv4"`
	SubnetCIDR  string          `json:"subnet_cidr"`
	Template    string          `json:"template"`
	Cost        string          `json:"monthly_cost"`
	Project     string          `json:"project"`
	Domain      string          `json:"domain"`
	Zone        string          `json:"zone"`
	Region      string          `json:"region"`
	Tags        []string        `json:"tags"`
	Peers       []string        `json:"peers"`
	Kawaii      *KawaiiMetadata `json:"kawaii,omitempty"`
	Konvey      *KonveyMetadata `json:"konvey,omitempty"`
}
