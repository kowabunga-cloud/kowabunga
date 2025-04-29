/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package metadata

type KonveyMetadata struct {
	PrivateInterface     string                   `json:"private_interface"`
	VrrpControlInterface string                   `json:"vrrp_control_interface"`
	VirtualIPs           []VirtualIpMetadata      `json:"virtual_ips"`
	Endpoints            []KonveyEndpointMetadata `json:"endpoints"`
}

type KonveyEndpointMetadata struct {
	Name     string                  `json:"name"`
	Port     int64                   `json:"port"`
	Protocol string                  `json:"protocol"`
	Backends []KonveyBackendMetadata `json:"backends"`
}

type KonveyBackendMetadata struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
}
