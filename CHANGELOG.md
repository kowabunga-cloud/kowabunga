## unreleased (yyyy-mm-dd)

## 0.63.3 (2025-09-11)
* **BUG**: **kompute**: fix some private IP assignment issue when 'public' (exposed) network contains private subnets.

## 0.63.2 (2025-09-03)
* **NEW**: Add support for ARM64 architecture.
* **NEW**: Update build dependencies.
* **NEW**: updated gosec to v2.22.8.
* **NEW**: updated golangci-lint to v2.4.0.
* **BUG**: correct SMTP email format (html first, plain text as fallback)

## 0.63.1 (2025-05-08)
* **NEW**: updated logo in email notifications.
* **NEW**: updated dependencies.
* **BUG**: fix APT packages repo URL in Linux cloud-init.

## 0.63.0 (2025-05-02)
* **NEW**: **kahuna**: switched to MongoDB driver v2.
* **NEW**: **kawaii**: When creating a Kawaii Public and Private VIP, those are now coupled under a same Virtual Router ID
* **NEW**: **misc**: upgraded to golangci-lint v2, fixes compliance issues.
* **BUG**: **kawaii**: IPsecs routing is updated dynamically on VIP failover
* **BUG**: **kawaii**: add a firewall rule to allow AH and ESP protocols  in a tunnel

## 0.62.6 (2025-03-06)
* **NEW**: requires Go SDK 1.24.

## 0.62.5 (2025-03-06)
* **NEW**: implemented public API **v0.52.3**.

## 0.62.4 (2025-03-05)
* **NEW**: updated dependencies.
* **NEW**: updated gosec to v2.22.2.
* **NEW**: updated govulncheck to v1.1.4.
* **NEW**: updated golangci-lint to v1.64.6.
* **BUG**: **kaktus**: fix segmentation fault if downloaded QCOW image does not have additional headers field.
* **BUG**: **kahuna**: check for template name unicity per pool, not globally.

## 0.62.3 (2025-02-20)
* **NEW**: updated dependencies.
* **BUG**: **kawaii**: add forward rules on firewall to allow traffic between peered subnets

## 0.62.2 (2025-01-14)
* **NEW**: updated dependencies.
* **BUG**: **kahuna**: extend router permissions for non-admin users to query **region** and **zone** endpoints.

## 0.62.1 (2024-12-17)
* **NEW**: updated dependencies.
* **BUG**: **kawaii**: enforce proper OpenSWAN service reload at configuration change.

## 0.62.0 (2024-12-16)
* **NEW**: implemented public API **v0.52.1**.
* **NEW**: updated dependencies.
* **NEW**: **kawaii**: added support IPsec features (strongswan managed-app backend).
* **NEW**: **kawaii**: updated metadata scheme.
* **NEW**: **cloud-init**: extended Windows template with NuGet, Paket and PsExec packages installation.
* **BUG**: **cloud-init**: ensure proper Windows admin password setting.
* **BUG**: **kowarp**: fixed various linting issues.

## 0.61.0 (2024-10-17)
* **NEW**: implemented 2-steps user password recovery for security purpose.
* **NEW**: implemented new user session logout and self password reset API calls.
* **NEW**: export user role information in JWT session token (required for Koala Web UI).

## 0.60.1 (2024-10-11)
* **BUG**: retagged to cope with Debian versionning issue.

## 0.60.0 (2024-10-11)
* **NEW**: improved generated emails and add custom theme support.
* **BUG**: fix SDK server-side code generation (issue with objects nested required parameters).
* **BUG**: fixed Debian packages publishing issues.

## 0.60.0-rc1 (2024-10-10)
* **BREAKING CHANGE**: update to new v0.50.0 API, major resources renaming: KGW to Kawaii, KFS to Kylo, KCE to Kompute, NetGW to Kiwi and Pool to StoragePool.
* **BREAKING CHANGE**: updated database schema, requires documents migration.
* **NEW**: implemented database document migration helpers and per-collection schema versioning.
* **NEW**: implemented new **--migrate** command-line flag to perform live MongoDB collections and documents migration.
* **NEW**: restructured the whole source code tree.
* **NEW**: server-side SDK is now directly built-in and auto-generated, instead of using external one, allows for easier development version pinning at engineering stage.
* **NEW**: updated dependencies.
* **BUG**: **cloud-init**: fix public network adapter gateway.

## 0.51.0 (2024-09-17)
* **NEW**: extended **common** library with **DownloadFromURL()** method to efficiently retrieve remote files from HTTP.
* **NEW**: extended **agent** library with ZSTD stream decompression routine.
* **NEW**: extended **agent** library with QCOW2 image support and raw-format conversion.
* **NEW**: extended **ceph** plugin to use new resource download framework and QCOW2 to RAW disk image conversion.
* **NEW**: updated build and runtime dependancies.

## 0.50.6 (2024-09-13)
* **BUG**: fix KGW NAT routing from private LAN traffic (network loop).

## 0.50.5 (2024-09-09)
* **NEW**: add Makefile **tests** directive.
* **NEW**: added new official infographics sources.
* **NEW**: added tests for Konvey agent templating.
* **BUG**: fix test-suite, now passes successfully.
* **BUG**: fix **konvey** traefik configuration templating.

## 0.50.4 (2024-08-29)
* **NEW**: extended Prometheus metrics with instance-based information.
* **BUG**: fix **kgw** nftables + keepalived configuration templating.

## 0.50.3 (2024-08-28)
* **BUG**: **konvey**: ensure valid Traefik configuration settings.
* **BUG**: **konvey**: enforce cross-hosts selection on failover deployments.
* **BUG**: **konvey**: correctly use provided resource name.
* **BUG**: **kgw**: ensure nf_conntrack kernel driver is properly loaded.

## 0.50.2 (2024-08-27)
* **NEW**: upgraded compiler requirement to Go 1.23.
* **NEW**: updated build and runtime dependancies.
* **BUG**: add missing **konvey** resource name in JSON model output, fixes Terraform state inconsistencies.
* **BUG**: tune-in **kgw** NetFilter conntrack settings.

## 0.50.1 (2024-08-09)
* **NEW**: Added support for public API v0.42.
* **NEW**: Added **Konvey** Layer-4 Network Load-Balancer service.
* **NEW**: Restored internal HAR (Highly Available Resource) support to create failover service instances.
* **NEW**: Extended Zone and Regions internal APIs to guess for best-suited computing hosts.
* **NEW**: Add **kontrollers**: a new type of built-in agent for as-a-service instances, replacing Ansible to auto-configure system apps based on instance metadata. Live service update is triggered through WebSocket RPC notification.
* **NEW**: fully get rid of Ansible services post-provisionning.
* **BUG**: Fixed MZR proper project resources references deletion.
* **BUG**: relax **instance** clean-up, prevent from conflicting conditions and ghost DB objects.
* **BUG**: various dependancies upgrades and CVE fixes.

## 0.40.3 (2024-07-25)
* **NEW**: updated build and runtime dependancies.
* **NEW**: instance metadata now also displays underlying virtualization host identity
* **BUG**: instance metadata does not display KGW specific fields (even if null) on non-KGW instances.
* **BUG**: prometheus metrics now correctly reflect storage pool and project costs
* **BUG**: fix KNA PowerDNS error reflection code

## 0.40.2 (2024-07-11)
* **NEW**: KGW now support dynamic updates of firewall and NAT rules (no resource deletion/creation required anymore).
* **BUG**: whitelist KNA PowerDNS recursor zone creation 422 error code
* **BUG**: fix Windows-based instances CloudInit network configuration

## 0.40.1 (2024-06-28)
* **BUG**: fix KNA DNS zone creation test condition.

## 0.40.0.0 (2024-06-13)
* **NEW**: Updated to Kowabunga API v0.41.0.
* **NEW**: KCE instances now supports more than 2 network interfaces.
* **NEW**: Added new multi-zones-resource (MZR) meta-object for as-a-service instances spread over local zones for high-availability.
* **NEW**: KGW now uses MZR resources to provide true cross-zones redundancy, with per-zone private and public gateways and cross-vnet peerings. Fully automated, cross-networks routing, traffic firewalling and forwarding, and public DNAT port-forwarding.
* **NEW**: down-sized KGW hardware requirements.
* **NEW**: KGW now uses dynamic instance metadata information for self-provisionning instead of static cloudinit ones
* **NEW**: cloudinit metadata are now instructed with Kowabunga specifics
* **NEW**: added built-in **kw-meta** Linux utility to retrieve dynamic instance metadata from API.
* **NEW**: extended router header parsing for instance metadata API and improved query response times
* **NEW**: reserve project-specific local gateway IP addresses.
* **NEW**: add new runtime database schema migration and auto-pruner helpers.
* **NET**: add routines to virtual network resources to find the most appropriate subnet, with enough room to host large services.
* **NEW**: KNA agent now bypasses Iris middleware and used direct connection to Iris' PowerDNS APIs.
* **NEW**: project now provides list of VRRP IDs used by -as-a-service instances, not to be reused by end-user.
* **BUG**: instance and KCE's libvirt generated XML now ensures that network and disk interfaces are sorted in the right order and immutable.
* **BUG**: more robust and resilient resources deletion in case of missing objects or cross-references.

## 0.30.0.0 (2024-05-07)
* **NEW**: Stable v0.30 release

## 0.30.0.0-rc4 (2024-04-16)
* **BUG**: fix pool object database value override at each scan period.
* **BUG**: fix instance update and XML generation.
* **BUG**: fix volume BSON parsing.
* **BUG**: implemented API v0.31 changes; project volume creation depends on region, not zone.
* **BUG**: drastically increase http/wsrpc timeout (support for large templates upload)
* **BUG**: add fallback vlan gateway if no kgw in subnet
* **BUG**: fix users assignment in groups.
* **BUG**: move nfs ganesha api backends management from core to KSA agent.
* **BUG**: kgw cloudinit now uses the right public gateway, not an hardcoded one.
* **NEW**: updated dependancies.

## 0.30.0.0-rc3 (2024-04-10)
* **BUG**: fix KCA memory calculation segv on some particular NUMA architectures.
* **BUG**: updated dep to fix HTTP2 CVE

## 0.30.0.0-rc2 (2024-04-05)
* **BUG**: fix some CI/CD build issues
* **BUG**: force user password and API keys generation not to use symbols that might break stupid JSON/YAML parsers.

## 0.30.0.0-rc1 (2024-03-28)
* **BREAK**: Major public API update
    * Migrated to OpenAPI v3.1.
    * Rely on brand new server-side SDK (every routing engine parts have been replaced).
    * Deprecation of **storage pool** types: **local** capability is gone, **ceph** becomes the only supported backend.
    * Update of **template** resources API, Ceph OS volumes are now automatically created from source HTTP(S) URL.

* **BREAK**: Proper Multi-AZ service readiness
    * **Storage Pools** and **NFS** are now part of region, cross-zones, not zone-bounded anymore.
    * **Virtual Networks** and **Subnets** are now part of region, cross-zones, not zone-bounded anymore.
    * **KFS** and **KGW** as-as-sevices resources are now region-global, not zone-bounded anymore.
    * Resources from all zones should be able to use region's global services.

* **BREAK**: Revamped architecture
    * Introduction of n-tier architecture with **Kowabunga Agents**:
        * **KCA**: *Kowabunga Computing Agent*, locally controlling KVM/libvirt hypervisors.
        * **KNA**: *Kowabunga Networking Agent*, locally controlling Iris network gateways.
        * **KSA**: *Kowabunga Storage Agent*, locally controlling Ceph clusters.
    * Kowabunga is now split between the global, Internet-exposed orchestrator, and datacenter-local agent (KCA, KNA, KSA) instances, managing private local resources. Agents connect to orchestrator through secure WebSocket (bypassing all possible private-DC firewall issues) and gets controlled by reverse-RPC calls.
    * Typical usage workflow translates as **Terraform <-> Client SDK <-> API <-> Kowabunga Orchestrator <-> RPC <-> WSS <-> Local Agent**.
    * Both agents and orchestrator auto-detects peer's failure and automatically reconnect in a progressive manner.
    * All direct interaction from orchestrator to libvirt/Ceph/Iris is now delegated to respective agents.
    * Cloud-init ISO images are now stored on Ceph backend (distributed, ready for instances' migration) and do not require a patched version of libvirt anymore.
    * The libvirt XML resource schema generation has been fully refactored.
    * Orchestrator now features a in-memory ultra-efficient database cache (zero-copy, no garbage collection)

* **NEW**: Introduced user management features
    * Kowabunga now features users and groups of users.
    * Projects (and associated underlying resources) do not belong to anyone anymore and can only be created by users with **superAdmin** or **projectAdmin** role.
    * Users belong to one to several groups.
    * Projects are associated with groups.
    * All individuals from project's groups are allowed to access and administrate project's resources.
    * Orchestrator provides robust server-side generated API keys and user passwords, preventing user from using weak credentials
    * User are required to perform a 2-steps account validation upon creation, before being able to consume services.

* **NEW**: Introduced robust authentication mechanisms
    * Kowabunga features 3 ways to consume WebServices:
        * **admin master token** (should never be used, unless for creating first **superAdmin** users)
        * Server-to-server API key authentication
        * JWT-based bearer authentication
    * Orchestrator's HTTP router now features middleware-based API routing layers: log, authentication, authorization, processing
    * Orchestrator's HTTP router uses per API route ACLs checks.

* **NEW**: Miscellaneous features
    * New modular debian packaging, differentiating orchestrator from agents, with multi-architecture support (x86_64, arm64 ...)
    * Enforced Go 1.22 compiler
    * Fixes all known CVEs (to date)
    * Modularized Go packages with dynamic plugin support
    * Support for MacOS targets

* **BUG**: Ensure proper hardware stop at KCE's deletion: ensure proper volume erasure

## 0.10.1.1
* **NEW**: Updates default route to KGW

## 0.10.1.0
* **NEW**: ability to update KGW in a HA manner

## 0.10.0.1
* **BUG**: add missing project's KGW listing API call prototype.

## 0.10.0.0
* **NEW**: implemented project cost retrieval API
* **NEW**: implemented instance remote connection URL retrieval API
* **NEW**: implemented new special /latest/meta-data instance metadata API (AWS-style) to be queried by live instances to retrieve configuration properties
* **NEW**: re-implemented cost management API implementation, every resources now has its own price, flagged in DB.
* **NEW**: major DB queries optimization, fast and furious (should always have been done this way)
* **NEW**: Introduced ansible vars to be injected in cloud init template
* **NEW**: Implemented KGW (**Kowabunga Network Gateway**) object, a network gateway as-a-service, providing Internet inbound and outbound traffic to your project.

## 0.9.0.0
* **NEW**: updated Go compiler requirement to 1.21
* **NEW**: updated Kowabunga API to v0.8.0.
* **NEW**: updated dependancies against known CVEs.
* **NEW**: implemented NFS storage definition and KFS resources (Kowabunga File System, NFS-compatible shares).

## 0.8.0.0
* **NEW**: added support for pre-release binary vulnerability check (**make vuln**)
* **NEW**: extended Spice's virtual machine remote-display configuration to bind on all host interfaces
* **NEW**: add support for Windows-OS virtual machines
* **BREAK**: updated configuration file YAML syntax

## 0.7.8.1
* **BUG**: fix API handler web services

## 0.7.8.0
* **NEW**: expose Kowabunga metrics through native Prometheus format (/metrics endpoint)
* **NEW**: new custom HTTP server implementation, allows for multiple custom endpoint handlers
* **BUG**: cloud-init network config shuld not configure DNS settings if adapter doesn't have any associated IP address.

## 0.7.7.0
* **NEW**: implemented dns_record support from API v0.7.7
* **BUG**: only add instance private IP addresses to the internal DNS record

## 0.7.6.7
* **BUG**: fix possible ssh pubkey misformatting at cloud-init generation

## 0.7.6.6
* **BUG**: fix infinite round up for max score calculation on x86_64 archs

## 0.7.6.5
* **BUG**: extra sanity checks on host

## 0.7.6.4
* **BUG**: fix subnet's reserved range model generation

## 0.7.6.3
* **NEW**: improved host instance election algorithm
* **BUG**: prevent some possible nil pointer de-referencing
* **BUG**: fix multi-hosts spreading, ensuring local pool access is done from the host it belongs to

## 0.7.6.2
* fix deb scripts

## 0.7.6.1
* pre/post tasks on Debian packaging

## 0.7.6
* server-side implementation of Kowabunga API v0.7.6

## 0.1
* initial release
