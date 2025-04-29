/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package konvey

import (
	"testing"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents/templates"
)

const (
	TestKonveyServicesConfigDir = "/tmp/kowabunga/konvey"
)

var testKonveyServices = map[string]*agents.ManagedService{
	"keepalived": {
		BinaryPath: "",
		UnitName:   "keepalived",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.KeepalivedConfTemplate("konvey"),
				TargetPath:      "keepalived.conf",
			},
		},
	},
	"traefik": {
		BinaryPath: "",
		UnitName:   "traefik",
		ConfigPaths: []agents.ConfigFile{
			{
				TemplateContent: templates.TraefikConfTemplate("konvey"),
				TargetPath:      "traefik.yml",
			},
			{
				TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "tcp"),
				TargetPath:      "tcp.yml",
			},
			{
				TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "udp"),
				TargetPath:      "udp.yml",
			},
		},
	},
}

var testKonveyConfig = map[string]any{
	"konvey": map[string]any{
		"private_interface":      "ens4",
		"vrrp_control_interface": "ens4",
		"virtual_ips": []map[string]any{
			{
				"vrrp_id":   1,
				"interface": "ens4",
				"vip":       "192.168.0.10",
				"priority":  100,
				"mask":      24,
				"public":    false,
			},
		},
		"endpoints": []map[string]any{
			{
				"name":     "proxyServer",
				"port":     8080,
				"protocol": "tcp",
				"backends": []map[string]any{
					{
						"host": "192.168.0.20",
						"port": 8080,
					},
					{
						"host": "192.168.0.21",
						"port": 8080,
					},
				},
			},
		},
	},
}

func TestKonveyemplate(t *testing.T) {
	agents.AgentTestTemplate(t, testKonveyServices, TestKonveyServicesConfigDir, testKonveyConfig)
}
