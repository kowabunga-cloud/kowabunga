/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

import (
	"fmt"
)

const KeepalivedConfGoTmpl string = `
{{- with .%s }}
global_defs {
  vrrp_garp_master_refresh 1
}

  {{- $obj := .}}
  {{- $vrIDs := list }}
  {{- range $idx, $vip := .virtual_ips }}
    {{- $newVRID := not (has $vip.vrrp_id $vrIDs) }}
    {{- if $newVRID }}
    {{- $vrIDs = append $vrIDs $vip.vrrp_id }}
    {{- end }}

    {{- if $newVRID }}
vrrp_instance VI_{{ $vip.vrrp_id }} {
  interface {{ $obj.vrrp_control_interface }}
  state MASTER
  virtual_router_id {{ $vip.vrrp_id }}
  priority {{ $vip.priority }}
  # use_vmac vrrp{{ $vip.vrrp_id }}
  # vmac_xmit_base # transmit VRRP adverts over physical interface
  advert_int 1
  notify /etc/keepalived/notify.sh
  virtual_ipaddress {
      {{- range $_, $current_vip := $obj.virtual_ips }}
        {{- if eq $vip.vrrp_id  $current_vip.vrrp_id }}
    {{ $current_vip.vip }}/{{ $current_vip.mask }} dev {{ $current_vip.interface }}
        {{- end }}
      {{- end }}
  }

      {{ $hasPublic := false }}
      {{- range $_, $current_vip := $obj.virtual_ips }}
        {{- if and (eq $vip.vrrp_id  $current_vip.vrrp_id) ($current_vip.public) }}
          {{ $hasPublic = true }}
        {{- end }}
      {{- end }}
        {{- if $hasPublic }}
  virtual_routes {
    0.0.0.0/0 via {{ $obj.public_gw_address }} dev {{ $obj.public_interface }} metric {{ add 15 (mul 10 $idx) }}
  }
        {{- end }}
}
    {{- end }}
  {{- end }}
{{- end }}
`

const KeepalivedNotifyGoTmpl string = `#!/usr/bin/env bash
systemctl restart kowabunga-kawaii-agent strongswan
`

func KeepalivedConfTemplate(obj string) string {
	return fmt.Sprintf(KeepalivedConfGoTmpl, obj)
}
