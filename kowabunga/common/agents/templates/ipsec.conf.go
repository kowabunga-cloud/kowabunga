/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

const IPsecSwanctlConfGoTmpl string = `
{{- $root := . }}
connections {
{{- range $idx, $ipsec := .kawaii.ipsec_connections }}
    {{ $ipsec.name }} {
        local_addrs={{ $ipsec.ip }}
        remote_addrs={{ $ipsec.remote_peer }}
        local {
            auth=psk
            id={{ $ipsec.ip }}
        }
        remote {
            auth=psk
            id={{ $ipsec.remote_peer }}
        }
        version=2
        dpd_delay=5
        dpd_timeout={{ $ipsec.dpd_timeout }}
        rekey_time={{ $ipsec.rekey }}
        reauth_time={{ $ipsec.phase1_lifetime }}
        proposals={{ $ipsec.phase1_encryption_algorithm | lower }}-{{ $ipsec.phase1_integrity_algorithm | lower }}-{{ $ipsec.phase1_df_group }}
        if_id_in={{ $ipsec.xfrm_id }}
        if_id_out={{ $ipsec.xfrm_id }}
        children {
            {{ $ipsec.name }}_phase2 {
                local_ts={{ $root.subnet_cidr }}
                remote_ts={{ $ipsec.remote_subnet }}
                esp_proposals={{ $ipsec.phase2_encryption_algorithm | lower }}-{{ $ipsec.phase2_integrity_algorithm | lower }}-{{ $ipsec.phase2_df_group }}
                rekey_time={{ $ipsec.rekey }}
                life_time={{ $ipsec.phase2_lifetime }}
                dpd_action={{ $ipsec.dpd_action }}
                start_action={{ $ipsec.start_action }}
                mode=tunnel
            }
        }
   }
{{- end }}
}
secrets {
{{- range $idx, $ipsec := .kawaii.ipsec_connections }}
    ike_{{ $ipsec.name }} {
        id-1={{ $ipsec.ip }}
        id-2={{ $ipsec.remote_peer }}
        secret={{ $ipsec.pre_shared_key }}
    }
{{- end }}
}
`

const IPsecCharonGoTmpl string = `
charon {
    install_routes = no
    crypto_test {
    }
    host_resolver {
    }
    leak_detective {
    }
    processor {
        priority_threads {
        }
    }
    start-scripts {
    }
    stop-scripts {
    }
    tls {
    }
    x509 {
    }
}


`

const IPsecCharonLoggingGoTmpl string = `
charon {
    # two defined file loggers
    filelog {
        charon {
            # path to the log file, specify this as section name in versions prior to 5.7.0
            path = /var/log/charon.log
            # add a timestamp prefix
            time_format = %b %e %T
            # prepend connection name, simplifies grepping
            ike_name = yes
            # overwrite existing files
            append = no
            # increase default loglevel for all daemon subsystems
            default = 1
            # flush each line to disk
            flush_line = yes
        }
        stderr {
            # more detailed loglevel for a specific subsystem, overriding the
            # default loglevel.
            ike = 2
            knl = 2
        }
    }
    # and two loggers using syslog
    syslog {
        # prefix for each log message
        identifier = charon-custom
        # use default settings to log to the LOG_DAEMON facility
        daemon {
        }
        # very minimalistic IKE auditing logs to LOG_AUTHPRIV
        auth {
            default = -1
            ike = 0
        }
    }
}
`
const IPsecCharonLogrotateGoTmpl string = `
/var/log/charon.log {

    size 50M
    rotate 7
    compress

    delaycompress
    missingok
    postrotate
    endscript

    # If fail2ban runs as non-root it still needs to have write access
    # to logfiles.
    # create 640 fail2ban adm
    create 640 root adm
}

`
