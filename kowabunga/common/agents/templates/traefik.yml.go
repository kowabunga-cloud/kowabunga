/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

import (
	"fmt"
)

const (
	TraefikPingPort    = 8081
	TraefikMetricsPort = 8082
)

const TraefikGoTmpl string = `
{{ $root := .}}
{{- with .%s }}
global:
  checkNewVersion: false
  sendAnonymousUsage: false

entryPoints:
  {{- range $idx, $ep := .endpoints }}

  {{ $ep.name }}:
    address: :{{ $ep.port }}/{{ $ep.protocol }}

  {{- end }}

  ping:
    address: {{ index $root "local-ipv4" }}:%d
  metrics:
    address: {{ index $root "local-ipv4" }}:%d

# /ping
ping:
  entryPoint: "ping"

# /metrics
metrics:
  prometheus:
    entryPoint: metrics
    addEntryPointsLabels: true
    addRoutersLabels: true
    addServicesLabels: true

log:
 level: ERROR
 filePath: /var/log/traefik/traefik.log
 format: json

api:
  insecure: true
  dashboard: true

providers:
  file:
    directory: "/etc/traefik/conf.d"
    watch: true
{{- end }}
`

func TraefikConfTemplate(obj string) string {
	return fmt.Sprintf(TraefikGoTmpl, obj, TraefikPingPort, TraefikMetricsPort)
}
