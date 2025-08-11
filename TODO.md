# Kowabunga Services

- Core Systems:
  - **(Big) Kahuna** (Hawaiian - expert, the most dominant thing) => orchestrator
KVM HCI Nodes - Kador
  - **Kaktus** (Kowabunga Affordable KVM and Tight Underneath Storage): HCI node 
  - **Kiwi**: Kowabunga Inner Wan Interface
  - **Koala**: Web UI
- as-a-service:
  - **Kompute**: virtual machines
  - **Kylo**: distributed file system
  - **Kawaii** (Kowabunga Adaptive WAn Individual/Intelligent Interface): Internet gateway
  - **Konvey**: TCP LB
  - **Kalipso**: Application LB
  - **Karamail**: SMTP
  - **KissKool**
  - **Kanet**
  - **Kosta** (Slavic - reliability and trustworthiness, "data at rest"): Object storage - MinIO over RBD
  - **Kryo** (cold storage / backup ?) - logo: snowflake
  - **Kaddie**: PKI (step-ca ?)
  - **Kahuete** & **Kapero**: easter-eggs payload easter-eggs on API calls
  - **Kuagamole**
  - **Knox** (secrets mgr, openbao) (Knox Knox Knox Penny) - logo: safe
  - **Kaizen**
  - **Kratos**

# TODOs

## Kahuna
- Authentication:
  - [Goth](https://github.com/markbates/goth)
  - [Hanko](https://www.hanko.io/)
  - Local user database vs. OpenID integration.
- API:
  - Support for VRID self-registration.
  - Add **/kahouette**: Kowabunga being stateless, you won't get any cookie so let's have a peanut instead. JSON output of peanut key + ascii art value of a peanut or plain text.
  - Add **/organization** for multi-tenancy with user admin email (local auth) and possible OpenID integration (all users from a given org will use org's defined authentication scheme).
  - Auto rebalance: live migration (manual, auto)
  - Handling host maintenance: no schedule, movable workload, on/off to prevent workload auto-rescheduling
  - Add anti-affinity flag to instance, to prevent host collocation
  - Add Web service for /instance/id/migrate Action=plan | commit&Host=auto | id &live=true | false (migrate VM from one host to another (least used, best score) from same AZ)
  - Create VLAN's on demand on Kiwi + Kaktus when creating a new project: stop with the unmaintainable hardcoded list
  - host: Add 'eligible' flag to accept new workload
  - host: Add ping-of-death WS timeout to auto-reschedule workload

  
- Database:
  - Minimize DB calls (stop passing object id, reuse SQL connections)
- "Dev Mode" for contributors w/ single-node sandbox image sandbox
- [GWS](https://github.com/lxzan/gws) Web Socket server implementation replacement ?
    
## Kiwi
- **OPNSense** replacement: routing, WireGuard, OpenVPN, IPSEC, Firewall, NAT, BGP, OSPF
- Micro footprint OS, prebuilt images on GitHub, with single binary accepting YAML (for tests) or reloading versioned config every X seconds from Kowabunga orchestrator API with network/nftables config and auto-reload, connection args to be retrieve from kernel cmdline (based on [Alpine](https://www.alpinelinux.org/) or [FlatCar](https://www.flatcar.org/))
- Provide a multi-dc or poly-cloud federation schema with SDN mesh

## Kaktus
- Plugin module for local filesystem support (instead of Ceph), easier for dev home labs.

## Koala
- [Buffalo](https://github.com/gobuffalo/buffalo) – Rapid Web Development in Go
- [WeTTY](https://github.com/butlerx/wetty) Web terminal
- [WebSockify](https://github.com/msquee/go-websockify) to create WebSocket binding to each VM instance Spice port and use [Spice Web Client](https://github.com/eyeos/spice-web-client)
- Add .rdp file generation for simple Windows machines remote connect

## Kawaii
- Manage auto update

## Kylo
- Switch from external NFS Ganesha to Ceph-backed integration with subvolumes

## Karamail
- [Maddy](https://github.com/foxcpp/maddy?tab=readme-ov-file)
- [chasquid](https://blitiri.com.ar/p/chasquid/)

## Kaddie
- [Step CA](https://github.com/smallstep/certificates)

## Knox
- [Knox](https://github.com/pinterest/knox)
- [OpenBAO](https://github.com/openbao/openbao)

# Marketing

- **Message**: *No AI. No ML. No BS ! Simple purpose done right.*
- **WebSite**:
  - get inspired from **Caddy** WebServer
  - Add merchandising section
  - Add GitHub sponsoring section
- FOSDEM 2026 Talk ?
- LF or CNCF Integration








