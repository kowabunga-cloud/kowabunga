/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"sort"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionKawaiiSchemaVersion = 2
	MongoCollectionKawaiiName          = "kawaii"
	InternetGatewayCpu                 = 1
	InternetGatewayMemory              = 1 * common.GiB  // 4GB
	InternetGatewayDisk                = 16 * common.GiB // 16GB
	// Kawaii name will always result in kawaii-<regionname>
	KawaiiDefaultNamePrefix = "kawaii"

	KawaiiFirewallPolicyAccept    = "accept"
	KawaiiFirewallPolicyDrop      = "drop"
	KawaiiFirewallProtocolTCP     = "tcp"
	KawaiiFirewallProtocolUDP     = "udp"
	KawaiiFirewallWildcardNetwork = "0.0.0.0/0"
	KawaiiFirewallDirectionIn     = "in"
	KawaiiFirewallDirectionOut    = "out"
)

type Kawaii struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`

	// properties
	PublicSubnetID string             `bson:"public_subnet_id"`
	Firewall       KawaiiFirewall     `bson:"firewall"`
	DNatRules      []KawaiiDNatRule   `bson:"dnat_rules"`
	VpcPeerings    []KawaiiVpcPeering `bson:"vpc_peerings"`

	// children references
	MultiZonesResourceID string   `bson:"mzr_id"`
	IPsecIDs             []string `bson:"ipsec_ids"`
}

func KawaiiMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("kgws", MongoCollectionKawaiiName)
	if err != nil {
		return err
	}

	for _, kawaii := range FindKawaiis() {
		if kawaii.SchemaVersion == 0 || kawaii.SchemaVersion == 1 {
			err := kawaii.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type KawaiiFirewall struct {
	Ingress      []KawaiiFirewallIngressRule `bson:"ingress"`
	Egress       []KawaiiFirewallEgressRule  `bson:"egress"`
	EgressPolicy string                      `bson:"egress_policy"`
}

func (fw *KawaiiFirewall) Model() sdk.KawaiiFirewall {
	ingress := []sdk.KawaiiFirewallIngressRule{}
	for _, rule := range fw.Ingress {
		ingress = append(ingress, rule.Model())
	}

	egress := []sdk.KawaiiFirewallEgressRule{}
	for _, rule := range fw.Egress {
		egress = append(egress, rule.Model())
	}

	return sdk.KawaiiFirewall{
		Ingress:      ingress,
		EgressPolicy: fw.EgressPolicy,
		Egress:       egress,
	}
}

type KawaiiFirewallIngressRule struct {
	Source   string `bson:"source"`
	Protocol string `bson:"Protocol"`
	Ports    string `bson:"Ports"`
}

func (rule *KawaiiFirewallIngressRule) Model() sdk.KawaiiFirewallIngressRule {
	return sdk.KawaiiFirewallIngressRule{
		Source:   rule.Source,
		Protocol: rule.Protocol,
		Ports:    rule.Ports,
	}
}

type KawaiiFirewallEgressRule struct {
	Destination string `bson:"destination"`
	Protocol    string `bson:"protocol"`
	Ports       string `bson:"ports"`
}

func (rule *KawaiiFirewallEgressRule) Model() sdk.KawaiiFirewallEgressRule {
	return sdk.KawaiiFirewallEgressRule{
		Destination: rule.Destination,
		Protocol:    rule.Protocol,
		Ports:       rule.Ports,
	}
}

type KawaiiDNatRule struct {
	PrivateIP string `bson:"private_ip"`
	Protocol  string `bson:"protocol"`
	Ports     string `bson:"ports"`
}

func (rule *KawaiiDNatRule) Model() sdk.KawaiiDNatRule {
	return sdk.KawaiiDNatRule{
		Destination: rule.PrivateIP,
		Protocol:    rule.Protocol,
		Ports:       rule.Ports,
	}
}

type KawaiiVpcPeering struct {
	SubnetID string                 `bson:"subnet"`
	Policy   string                 `bson:"policy"`
	Ingress  []KawaiiVpcForwardRule `bson:"ingress"`
	Egress   []KawaiiVpcForwardRule `bson:"egress"`
	NetIP    []KawaiiVpcNetIpZone   `bson:"netip"`
}

func (kvp *KawaiiVpcPeering) Model() sdk.KawaiiVpcPeering {
	ingress := []sdk.KawaiiVpcForwardRule{}
	for _, rule := range kvp.Ingress {
		ingress = append(ingress, rule.Model())
	}

	egress := []sdk.KawaiiVpcForwardRule{}
	for _, rule := range kvp.Egress {
		egress = append(egress, rule.Model())
	}

	return sdk.KawaiiVpcPeering{
		Subnet:  kvp.SubnetID,
		Policy:  kvp.Policy,
		Ingress: ingress,
		Egress:  egress,
		Netip:   []sdk.KawaiiVpcNetIpZone{},
	}
}

type KawaiiVpcForwardRule struct {
	Protocol string `bson:"protocol"`
	Ports    string `bson:"ports"`
}

func (rule *KawaiiVpcForwardRule) Model() sdk.KawaiiVpcForwardRule {
	return sdk.KawaiiVpcForwardRule{
		Protocol: rule.Protocol,
		Ports:    rule.Ports,
	}
}

type KawaiiVpcNetIpZone struct {
	Zone      string `bson:"zone"`
	PrivateIP string `bson:"private_ip"`
}

func (net *KawaiiVpcNetIpZone) Model() sdk.KawaiiVpcNetIpZone {
	return sdk.KawaiiVpcNetIpZone{
		Zone:    net.Zone,
		Private: net.PrivateIP,
	}
}

func NewKawaii(projectId, regionId, name, desc string, fw KawaiiFirewall, dnat []KawaiiDNatRule, vpcPeerings []KawaiiVpcPeering) (*Kawaii, error) {

	// find parent objects, allows to bail before creating anything
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	region, err := FindRegionByID(regionId)
	if err != nil {
		return nil, err
	}
	vnets, err := region.FindVNets()
	if err != nil {
		return nil, err
	}

	// we request at least one public gateway VIP per zone
	requestedIps := len(prj.ZoneGateways)

	var publicSubnet *Subnet
	for _, v := range vnets {
		if v.Private {
			continue
		}

		// find a public subnet with enough free IP addresses
		s, err := v.FindFreeSubnet(requestedIps)
		if err != nil {
			return nil, err
		}

		publicSubnet = s
		break
	}

	klog.Debug("Creating underlying HA resources for Kawaii")

	// extract subnets to create peering with
	subnetPeerings := []string{}
	for _, vp := range vpcPeerings {
		subnetPeerings = append(subnetPeerings, vp.SubnetID)
	}

	kawaii := Kawaii{
		Resource:       NewResource(name, desc, MongoCollectionKawaiiSchemaVersion),
		ProjectID:      projectId,
		PublicSubnetID: publicSubnet.String(),
		Firewall:       fw,
		DNatRules:      dnat,
		VpcPeerings:    vpcPeerings,
	}

	mzr, err := NewMultiZonesResource(projectId, regionId, name, desc, CloudinitProfileKawaii, kawaii.String(), InternetGatewayCpu, InternetGatewayMemory, InternetGatewayDisk, 0, publicSubnet.String(), subnetPeerings)
	if err != nil {
		return nil, err
	}
	kawaii.MultiZonesResourceID = mzr.String()

	klog.Debugf("Created new Kawaii %s", kawaii.String())
	_, err = GetDB().Insert(MongoCollectionKawaiiName, kawaii)
	if err != nil {
		return nil, err
	}

	// read project object back, as it's been updated
	prj, err = FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	// add Kawaii to project
	prj.AddKawaii(kawaii.String())

	return &kawaii, err
}

func FindKawaiis() []Kawaii {
	return FindResources[Kawaii](MongoCollectionKawaiiName)
}

func FindKawaiisByProject(projectId string) ([]Kawaii, error) {
	return FindResourcesByKey[Kawaii](MongoCollectionKawaiiName, "project_id", projectId)
}

func FindKawaiiByID(id string) (*Kawaii, error) {
	return FindResourceByID[Kawaii](MongoCollectionKawaiiName, id)
}

func FindKawaiiByName(name string) (*Kawaii, error) {
	return FindResourceByName[Kawaii](MongoCollectionKawaiiName, name)
}

func (k *Kawaii) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKawaiiName, k.ID, from, to)
}

func (k *Kawaii) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKawaiiName, k.ID, version)
}

func (k *Kawaii) migrateSchemaV2() error {
	err := k.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("public_subnet", "public_subnet_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("mzr", "mzr_id")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kawaii) Project() (*Project, error) {
	prj, err := FindProjectByID(k.ProjectID)
	if err != nil {
		return nil, err
	}
	return prj, nil
}

func (k *Kawaii) Update(desc string, fw KawaiiFirewall, dnat []KawaiiDNatRule) error {
	// TODO: in future, we'll need to support vnetPeerings update.
	// This requires update of instance XML description to create/remove network adapters.

	k.Description = desc
	k.Firewall = fw
	k.DNatRules = dnat
	k.Save()

	mzr, err := k.MZR()
	if err != nil {
		return nil // bypass error
	}

	for _, komputeId := range mzr.KomputeIDs {
		kompute, err := FindKomputeByID(komputeId)
		if err != nil {
			continue
		}

		i, err := kompute.Instance()
		if err != nil {
			continue
		}

		args := agents.KontrollerReloadArgs{}
		var reply agents.KontrollerReloadReply
		err = i.InstanceRPC("Reload", args, &reply)
		if err != nil {
			continue
		}
	}

	return nil
}

func (k *Kawaii) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKawaiiName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Kawaii) Delete() error {
	klog.Debugf("Deleting Kawaii %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	mzr, err := k.MZR()
	if err != nil {
		klog.Error(err)
		return err
	}
	err = mzr.Delete()
	if err != nil {
		return err
	}

	// Remove IPsec Childs (This should probably happens as because of the dependency path,
	// IPsec shall always be removed first by TF)
	ipsecs, err := FindIPsecByKawaii(k.String())
	if err != nil {
		klog.Errorf("Could Not find Kawaii IPsecs. Perhaps it was deleted ? Kawaii may leave ipsec orphans")
	}
	for _, ipsec := range ipsecs {
		err := ipsec.Delete()
		if err != nil {
			klog.Debugf("Could not delete ipsec child.")
		}
	}
	// remove kawaii's reference from parents
	prj, err := k.Project()
	if err != nil {
		return err
	}
	prj.RemoveKawaii(k.String())

	return GetDB().Delete(MongoCollectionKawaiiName, k.ID)
}

func (k *Kawaii) Model() sdk.Kawaii {
	kawaii := sdk.Kawaii{
		Id:          k.String(),
		Description: k.Description,
	}

	prj, err := k.Project()
	if err != nil {
		return kawaii
	}

	mzr, err := k.MZR()
	if err != nil {
		return kawaii
	}

	instances, err := k.InstanceIDs()
	if err != nil {
		return kawaii
	}

	// Network IP Settings
	netIP := sdk.KawaiiNetIp{}
	for zone, gw := range prj.ZoneGateways {
		netIP.Private = append(netIP.Private, gw)

		zs := sdk.KawaiiNetIpZone{
			Zone:    zone,
			Private: gw,
		}

		adapterId, ok := mzr.PublicAdapterIDs[zone]
		if ok {
			adapter, err := FindAdapterByID(adapterId)
			if err != nil {
				return kawaii
			}
			if len(adapter.Addresses) > 0 {
				zs.Public = adapter.Addresses[0]
				netIP.Public = append(netIP.Public, adapter.Addresses[0])
			}
		}

		netIP.Zones = append(netIP.Zones, zs)
	}
	sort.Strings(netIP.Public)
	sort.Strings(netIP.Private)
	kawaii.Netip = netIP

	// Firewall Settings
	kawaii.Firewall = k.Firewall.Model()

	// DNAT Settings
	dnat := []sdk.KawaiiDNatRule{}
	for _, nr := range k.DNatRules {
		dnat = append(dnat, nr.Model())
	}
	kawaii.Dnat = dnat

	// VPC Peerings
	vpcPeerings := []sdk.KawaiiVpcPeering{}
	for _, vp := range k.VpcPeerings {
		vpm := vp.Model()
		for _, instanceId := range instances {
			i, err := FindInstanceByID(instanceId)
			if err != nil {
				continue
			}
			for _, adapterId := range i.Adapters() {
				adapter, err := FindAdapterByID(adapterId)
				if err != nil {
					continue
				}

				k, err := i.Kaktus()
				if err != nil {
					continue
				}

				zone, err := k.Zone()
				if err != nil {
					continue
				}

				if adapter.SubnetID == vpm.Subnet {
					netip := KawaiiVpcNetIpZone{
						Zone:      zone.Name,
						PrivateIP: adapter.Addresses[0],
					}
					vpm.Netip = append(vpm.Netip, netip.Model())
				}
			}
		}
		vpcPeerings = append(vpcPeerings, vpm)
	}
	kawaii.VpcPeerings = vpcPeerings

	return kawaii
}

func (k *Kawaii) MZR() (*MultiZonesResource, error) {
	return FindMZRByID(k.MultiZonesResourceID)
}

func (k *Kawaii) InstanceIDs() ([]string, error) {
	var instances []string

	mzr, err := k.MZR()
	if err != nil {
		return instances, err
	}

	for _, komputeId := range mzr.KomputeIDs {
		kompute, err := FindKomputeByID(komputeId)
		if err != nil {
			return instances, err
		}
		instances = append(instances, kompute.InstanceID)
	}

	return instances, nil
}

func (k *Kawaii) Metadata(instanceId string) metadata.KawaiiMetadata {
	publicInterface := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex)
	privateInterface := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex+1)

	meta := metadata.KawaiiMetadata{
		PublicInterface:              publicInterface,
		PrivateInterface:             privateInterface,
		PeeringInterfaces:            []string{},
		VrrpControlInterface:         privateInterface,
		PublicVipAddresses:           []string{},
		VirtualIPs:                   []metadata.VirtualIpMetadata{},
		FirewallDefaultInputPolicy:   KawaiiFirewallPolicyDrop,
		FirewallDefaultOutputPolicy:  k.Firewall.EgressPolicy,
		FirewallDefaultForwardPolicy: KawaiiFirewallPolicyDrop,
		FirewallInputExtraNetworks:   []string{},
		FirewallInputRules:           []metadata.KawaiiFirewallRuleMetadata{},
		FirewallOutputRules:          []metadata.KawaiiFirewallRuleMetadata{},
		FirewallForwardRules:         []metadata.KawaiiFirewallRuleMetadata{},
		FirewallNatRules:             []metadata.KawaiiFirewallNatRuleMetadata{},
		IPsecConnections:             []metadata.KawaiiIPsecConnectionMetadata{},
	}

	for i := 0; i < len(k.VpcPeerings); i++ {
		dev := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex+2+i)
		meta.PeeringInterfaces = append(meta.PeeringInterfaces, dev)
	}

	publicSubnet, err := FindSubnetByID(k.PublicSubnetID)
	if err != nil {
		return meta
	}
	meta.PublicGateway = publicSubnet.Gateway

	mzr, err := k.MZR()
	if err != nil {
		return meta
	}

	privateSubnet, err := FindSubnetByID(mzr.PrivateSubnetID)
	if err != nil {
		return meta
	}
	meta.FirewallInputExtraNetworks = privateSubnet.Routes

	prj, err := k.Project()
	if err != nil {
		return meta
	}

	instance, err := FindInstanceByID(instanceId)
	if err != nil {
		return meta
	}

	ks, err := instance.Kaktus()
	if err != nil {
		return meta
	}

	zone, err := ks.Zone()
	if err != nil {
		return meta
	}

	meta.PublicVipAddresses = mzr.PublicVIPs

	for _, ip := range mzr.VirtualIPs {
		meta.VirtualIPs = append(meta.VirtualIPs, ip.Metadata())
	}

	// tune-in VRRP priority, per-zone affinity
	for id, vip := range meta.VirtualIPs {
		if vip.Public {
			adapter, err := FindAdapterByID(mzr.PublicAdapterIDs[zone.Name])
			if err != nil {
				continue
			}

			meta.VirtualIPs[id].Priority = VrrpPriorityBackup
			if vip.VIP == adapter.Addresses[0] {
				meta.VirtualIPs[id].Priority = VrrpPriorityMaster
			}
		} else {
			zoneGateway := "NONE"
			gw, ok := prj.ZoneGateways[zone.Name]
			if ok {
				zoneGateway = gw
			}

			meta.VirtualIPs[id].Priority = VrrpPriorityBackup
			if vip.VIP == zoneGateway {
				meta.VirtualIPs[id].Priority = VrrpPriorityMaster
			}
		}
	}

	for _, vip := range mzr.VirtualIPs {
		if !vip.Public {
			continue
		}

		// firewall input rules
		for _, in := range k.Firewall.Ingress {
			rule := metadata.KawaiiFirewallRuleMetadata{
				InputInterface:  vip.Interface,
				OutputInterface: vip.Interface,
				Source:          in.Source,
				Destination:     vip.VIP,
				Protocol:        in.Protocol,
				Ports:           in.Ports,
				Action:          KawaiiFirewallPolicyAccept,
			}
			meta.FirewallInputRules = append(meta.FirewallInputRules, rule)
		}

		// firewall output rules
		for _, out := range k.Firewall.Egress {
			rule := metadata.KawaiiFirewallRuleMetadata{
				OutputInterface: vip.Interface,
				Source:          KawaiiFirewallWildcardNetwork,
				Destination:     out.Destination,
				Protocol:        out.Protocol,
				Ports:           out.Ports,
				Action:          KawaiiFirewallPolicyDrop,
			}
			meta.FirewallOutputRules = append(meta.FirewallOutputRules, rule)
		}

		// firewall NAT rules
		for _, nr := range k.DNatRules {
			rule := metadata.KawaiiFirewallNatRuleMetadata{
				PrivateIP: nr.PrivateIP,
				PublicIP:  vip.VIP,
				Protocol:  nr.Protocol,
				Ports:     nr.Ports,
			}
			meta.FirewallNatRules = append(meta.FirewallNatRules, rule)
		}
	}

	// firewall forward rules
	vpcInterfaceIndex := AdapterOsNicLinuxStartIndex + 2
	for _, vpc := range k.VpcPeerings {
		vpcInterface := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, vpcInterfaceIndex)
		vpcSubnet, err := FindSubnetByID(vpc.SubnetID)
		if err != nil {
			continue
		}
		for _, in := range vpc.Ingress {
			rule := metadata.KawaiiFirewallRuleMetadata{
				InputInterface:  privateInterface,
				OutputInterface: vpcInterface,
				Source:          privateSubnet.CIDR,
				Destination:     vpcSubnet.CIDR,
				Protocol:        in.Protocol,
				Direction:       KawaiiFirewallDirectionOut,
				Ports:           in.Ports,
				Action:          KawaiiFirewallPolicyAccept,
			}
			// rule's policy is exception to default one
			if vpc.Policy == KawaiiFirewallPolicyAccept {
				rule.Action = KawaiiFirewallPolicyDrop
			}
			meta.FirewallForwardRules = append(meta.FirewallForwardRules, rule)
		}
		for _, out := range vpc.Egress {
			rule := metadata.KawaiiFirewallRuleMetadata{
				InputInterface:  vpcInterface,
				OutputInterface: privateInterface,
				Source:          vpcSubnet.CIDR,
				Destination:     privateSubnet.CIDR,
				Protocol:        out.Protocol,
				Direction:       KawaiiFirewallDirectionIn,
				Ports:           out.Ports,
				Action:          KawaiiFirewallPolicyAccept,
			}
			// rule's policy is exception to default one
			if vpc.Policy == KawaiiFirewallPolicyAccept {
				rule.Action = KawaiiFirewallPolicyDrop
			}
			// enforce application's ports if specified at subnet's level
			if vpcSubnet.Application == SubnetApplicationCeph {
				rule.Ports = SubnetApplicationCephPorts
			}
			meta.FirewallForwardRules = append(meta.FirewallForwardRules, rule)
		}
		vpcInterfaceIndex += 1
	}

	// Kawaii Ip Sec
	ipsecs := []metadata.KawaiiIPsecConnectionMetadata{}
	for _, ipsec := range k.IPsecIDs {
		ipsec, err := FindIPsecByID(ipsec)
		if err != nil {
			klog.Errorf("Kawaii Metadata : Could not find IPsec %s", ipsec)
		}
		ipsecMeta := ipsec.Metadata()
		ipsecs = append(ipsecs, *ipsecMeta)
	}
	meta.IPsecConnections = ipsecs
	return meta
}

func (k *Kawaii) AddIPsec(ipsecID string) {
	klog.Debugf("Adding IPsec %s to pool %s", ipsecID, k.String())
	AddChildRef(&k.IPsecIDs, ipsecID)
	k.Save()
}

func (k *Kawaii) RemoveIPsec(ipsecID string) {
	klog.Debugf("Removing IPsec %s from pool %s", ipsecID, k.String())
	RemoveChildRef(&k.IPsecIDs, ipsecID)
	k.Save()
}

func IPsecIngressRuleToMetadata(rule *KawaiiFirewallIngressRule) *metadata.KawaiiFirewallRuleMetadata {
	return &metadata.KawaiiFirewallRuleMetadata{
		Source:    rule.Source,
		Protocol:  rule.Protocol,
		Direction: KawaiiFirewallDirectionIn,
		Ports:     rule.Ports,
		Action:    KawaiiFirewallPolicyAccept,
	}
}

func (k *Kawaii) HasChildren() bool {
	return HasChildRefs(k.IPsecIDs)
}
