/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

const NftablesFirewallGoTmpl string = `
{{- define "createRules" }}
{{- range $idx, $rule :=  .templateRules }}
        {{- $portType := "dport" }}
        {{- if (ne $rule.direction "out") }}
        {{- $portType = "sport" }}
        {{- end }}
        iifname {{ $rule.iifname }}  oifname {{ $rule.oifname }} ip saddr {{ $rule.source_ip }} ip daddr {{ $rule.destination_ip }} {{ $rule.protocol }} {{ $portType }} { {{ $rule.ports }} } {{ $rule.action }}

        {{- if  $.isForward }}
                {{- if or (eq $rule.direction "out") (not $rule.direction) }}
                        {{- range $i, $vip :=  $.kawaii.virtual_ips }}
                                {{- if (eq $vip.public false) }}
        iifname vrrp{{ $vip.vrrp_id }}  oifname {{ $rule.oifname }}  ip saddr {{ $rule.source_ip }} ip daddr {{ $rule.destination_ip }} {{ $rule.protocol }} dport { {{ $rule.ports }} } {{ $rule.action }}
                                {{- end }}
                        {{- end }}
                {{- end }}
        {{- end }}
{{- end }}
{{- end }}

table inet firewall {
    chain global {
        ct state established,related accept
    }
    chain input {
        # Default {{ .kawaii.fw_input_default }}
        type filter hook input priority 0; policy {{ .kawaii.fw_input_default }};
        ct state established,related accept
        # allow vrrp
        iifname {{ .kawaii.vrrp_control_interface }} ip protocol vrrp accept
        # allow incoming traffic on the LAN interface
        iifname {{ .kawaii.private_interface }} accept
        # accept any localhost traffic
        iif lo accept

        # Makes sure DNS is ok
        tcp dport 53 accept
        udp dport 53 accept
        udp sport 53 accept

        #IPsec
{{- range $_, $ipsec := .kawaii.ipsec_connections }}
        iifname {{ $.kawaii.public_interface }} ip saddr {{ $ipsec.remote_peer }} udp dport {500, 4500} accept
        iifname {{ $.kawaii.public_interface }} ip saddr {{ $ipsec.remote_peer }} ip protocol { ah, esp } accept
{{- end }}

 #Ports validity should have been checked beforehand
{{- range $idx, $nat := .kawaii.fw_nat_rules }}
        ip saddr 0.0.0.0/0 tcp dport { {{ $nat.ports}} } accept
{{- end }}
        # VPN access
{{- range $idx, $cidr := .kawaii.fw_input_extra_networks }}
        ip saddr {{ $cidr }} tcp dport ssh accept;
{{- end }}
# customer rules
{{- $_ := set .  "isForward" false}}
{{- $_ := set .  "templateRules" .kawaii.fw_input_rules }}
{{- template "createRules" . }}

    }
    chain forward {
        type filter hook forward priority 0; policy {{ .kawaii.fw_forward_default }};

{{- $_ := set .  "isForward" true}}
{{- $_ := set .  "templateRules" .kawaii.fw_forward_rules }}
{{ template "createRules" . }}
# forward rules from  lan VIP to public
          iifname {{ .kawaii.public_interface }} oifname {{ .kawaii.private_interface }} accept
          iifname {{ .kawaii.private_interface }} oifname {{ .kawaii.public_interface }} accept
          iifname {{ .kawaii.private_interface }} oifname {{ .kawaii.private_interface }} accept

{{- range $i, $vip := .kawaii.virtual_ips }}
        {{- if (eq $vip.public false) }}
          iifname vrrp{{ $vip.vrrp_id }}  oifname {{ $.kawaii.public_interface }} accept
          iifname vrrp{{ $vip.vrrp_id }}  oifname {{ $.kawaii.private_interface }} accept
        {{- end }}
{{- end }}

        #############################
        # Peerings -allow traffic   #
        #############################
{{- range $idx, $peeringInterface := .kawaii.peering_interfaces }}
          iifname {{ $peeringInterface }}  oifname {{ $.kawaii.private_interface }} accept
          iifname {{ $.kawaii.private_interface  }}  oifname {{ $peeringInterface }} accept
{{- end }}

        ##############
        # XFRM IPSEC #
        ##############
{{- range $i, $ipsec := .kawaii.ipsec_connections }}
        {{- if $ipsec.ingress_rules }}
                {{ range $idx, $ing := $ipsec.ingress_rules }}
        iifname xfrm-{{ $ipsec.xfrm_id }} {{ $ing.protocol }} dport { {{ $ing.ports }} } accept
                {{- end }}
        {{- else }}
        iifname xfrm-{{ $ipsec.xfrm_id }} accept
        {{- end }}
        oifname xfrm-{{ $ipsec.xfrm_id }} accept
{{- end }}

    }

    chain output {
        type filter hook output priority 0; policy {{ .kawaii.fw_output_default }};
{{- $_ := set .  "isForward" false}}
{{- $_ := set .  "templateRules" .kawaii.fw_output_rules }}
{{- template "createRules" . }}
    }
}`
