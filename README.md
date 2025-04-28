<p align="center">
  <a href="https://www.kowabunga.cloud/?utm_source=github&utm_medium=logo" target="_blank">
    <picture>
      <source srcset="https://raw.githubusercontent.com/kowabunga-cloud/infographics/master/art/kowabunga-title-white.png" media="(prefers-color-scheme: dark)" />
      <source srcset="https://raw.githubusercontent.com/kowabunga-cloud/infographics/master/art/kowabunga-title-black.png" media="(prefers-color-scheme: light), (prefers-color-scheme: no-preference)" />
      <img src="https://raw.githubusercontent.com/kowabunga-cloud/infographics/master/art/kowabunga-title-black.png" alt="Kowabunga" width="800">
    </picture>
  </a>
</p>

# Kowabunga

This is **Kowabunga**, a complete infrastructure automation suite to orchestrate virtual resources management automation on privately-owned commodity hardware.

[![License: Apache License, Version 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://spdx.org/licenses/Apache-2.0.html)
[![go.dev](https://img.shields.io/badge/go.dev-pkg-007d9c.svg?style=flat)](https://pkg.go.dev/github.com/kowabunga-cloud/kowabunga)
[![Build Status](https://github.com/kowabunga-cloud/kowabunga/actions/workflows/go.yml/badge.svg)](https://github.com/kowabunga-cloud/kowabunga/actions/workflows/go.yml)

This repository features the server-side bits of Kowabunga, including:

- **Kahuna**, the orchestration system, which remotely controls every resource and maintains ecosystem consistent. Gateway to the Kowabunga REST API.
- **Kiwi** agent, for SD-WAN nodes. It provides various network services like routing, firewall, DHCP, DNS, VPN, IPSec peering (with active-passive failover).
- **Kaktus** agent, for HCI nodes. It supports virtual computing hypervisor with distributed storage services.

## Current Releases

| Project            | Release Badge                                                                                       |
|--------------------|-----------------------------------------------------------------------------------------------------|
| **Kowabunga**           | [![Kowabunga Release](https://img.shields.io/github/v/release/kowabunga-cloud/kowabunga)](https://github.com/kowabunga-cloud/kowabunga/releases) |

## License

Licensed under [Apache License, Version 2.0](https://opensource.org/license/apache-2-0), see [`LICENSE`](LICENSE).
