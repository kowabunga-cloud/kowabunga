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
	TraefikRouterRule                   = "HostSNI(`*`)"
	TraefikLoadBalancerTerminationDelay = 200
	TraefikLoadBalancerProxyProtocol    = 1
)

const TraefikLayer4GoTmpl string = `
{{- with .%s }}

{{ $enabled := false }}
{{- range $ep := .endpoints }}
{{ if eq .protocol "%s" }}
{{ $enabled = true }}
{{- end }}
{{- end }}

{{ if $enabled }}
%s:
  routers:
    {{- range $ep := .endpoints }}
    {{ if eq .protocol "%s" }}
    {{ $ep.name }}:
      entryPoints:
        - {{ $ep.name }}
      rule: "%s"
      service: "{{ $ep.name }}"
      tls:
        passthrough: true

    {{- end }}
    {{- end }}

  services:
    {{- range $ep := .endpoints }}
    {{ if eq .protocol "%s" }}
    {{ $ep.name }}:
      loadBalancer:
        terminationDelay: %d
        proxyProtocol: %d
        servers:
        {{- range $b := .backends }}
        - address: "{{ $b.host }}:{{ $b.port }}"
        {{- end }}

    {{- end }}
    {{- end }}
{{- end }}

{{- end }}
`

func TraefikLayer4ConfTemplate(obj, protocol string) string {
	return fmt.Sprintf(TraefikLayer4GoTmpl, obj, protocol, protocol, protocol,
		TraefikRouterRule, protocol, TraefikLoadBalancerTerminationDelay, TraefikLoadBalancerProxyProtocol)
}
