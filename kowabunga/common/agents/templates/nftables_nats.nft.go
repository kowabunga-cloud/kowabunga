/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

const NftablesNatsGoTmpl string = `
{{- $root := . }}
table nat {
    chain prerouting {
        type nat hook prerouting priority 100;

{{- range $idx, $natRule := .kawaii.fw_nat_rules }}
        iif {{ $.kawaii.public_interface }} ip daddr {{ $natRule.public_ip }} tcp dport { {{ $natRule.ports }} } dnat {{ $natRule.private_ip }}
        iif {{ $.kawaii.private_interface }} ip daddr {{ $natRule.public_ip }} tcp dport { {{ $natRule.ports }} } dnat {{ $natRule.private_ip }}
  {{- range $i, $vip := .kawaii.virtual_ips }}
        {{- if (eq $vip.public false) }}
        iifname vrrp{{ $vip.vrrp_id }}  oifname {{ $.kawaii.public_interface }} tcp dport { {{ $natRule.ports }} } dnat {{ $natRule.private_ip }}
        {{- end }}
  {{- end }}
{{- end }}

    }

    chain postrouting {
        type nat hook postrouting priority 100;
{{- range $idx, $ipsec := .kawaii.ipsec_connections }}
        ip saddr {{ $ipsec.remote_subnet }} ip daddr {{ $root.subnet_cidr }} accept
        ip daddr {{ $ipsec.remote_subnet }} ip saddr {{ $root.subnet_cidr }} accept
{{- end }}
        oif { {{ .kawaii.public_interface }}, {{ .kawaii.private_interface }} } masquerade
{{- range $idx, $peeringInterface := .kawaii.peering_interfaces }}
        oif {{ $peeringInterface }} masquerade
{{- end }}
    }
}
`
