/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package templates

const NftablesConfGoTmpl string = `
flush ruleset

include "/etc/nft-network/nats.nft";

include "/etc/nft-network/firewall.nft";
`
