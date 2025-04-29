/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package konvey

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents/templates"
)

var konveyServices = map[string]*agents.ManagedService{
	"keepalived": &agents.ManagedService{
		BinaryPath: "",
		UnitName:   "keepalived.service",
		User:       "root",
		Group:      "root",
		ConfigPaths: []agents.ConfigFile{
			agents.ConfigFile{
				TemplateContent: templates.KeepalivedConfTemplate("konvey"),
				TargetPath:      "/etc/keepalived/keepalived.conf",
			},
		},
	},
	"traefik": &agents.ManagedService{
		BinaryPath: "", //TODO: Later use for binary upgrade mgmt
		UnitName:   "traefik.service",
		User:       "traefik",
		Group:      "traefik",
		ConfigPaths: []agents.ConfigFile{
			agents.ConfigFile{
				TemplateContent: templates.TraefikConfTemplate("konvey"),
				TargetPath:      "/etc/traefik/traefik.yml",
			},
			agents.ConfigFile{
				TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "tcp"),
				TargetPath:      "/etc/traefik/conf.d/tcp.yml",
			},
			agents.ConfigFile{
				TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "udp"),
				TargetPath:      "/etc/traefik/conf.d/udp.yml",
			},
		},
	},
}

func Daemonize() error {
	return agents.KontrollerDaemon(konveyServices, []agents.KowabungaSysctlSetting{})
}
