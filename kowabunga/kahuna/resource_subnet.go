/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"

	"github.com/netdata/go.d.plugin/pkg/iprange"
	ipa "github.com/seancfoley/ipaddress-go/ipaddr"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionSubnetSchemaVersion = 2
	MongoCollectionSubnetName          = "subnet"

	ErrSubnetNoSuchAdapter   = "no such adapter in subnet"
	SubnetMaximumNetMaskSize = 16

	SubnetApplicationUser      = "user"
	SubnetApplicationCeph      = "ceph"
	SubnetApplicationCephPorts = "111,2049,3300,6789,6800-7568" // Ceph + NFS ports
)

type Subnet struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`
	VNetID    string `bson:"vnet_id"`

	// properties
	CIDR        string     `bson:"cidr"`
	Gateway     string     `bson:"gateway"`
	DNS         string     `bson:"dns"`
	Reserved    []*IPRange `bson:"reserved_ranges"`
	GwPool      []*IPRange `bson:"gw_pool"`
	Routes      []string   `bson:"routes"`
	Application string     `bson:"application"`

	// children references
	AdapterIDs []string `bson:"adapter_ids"`
}

type IPRange struct {
	First string `bson:"first"`
	Last  string `bson:"last"`
}

func (ipr *IPRange) Size() int {
	first := net.ParseIP(ipr.First)
	if first == nil {
		return 0
	}

	last := net.ParseIP(ipr.Last)
	if last == nil {
		return 0
	}

	rg := iprange.New(first, last)
	if rg == nil {
		return 0
	}

	size := rg.Size()
	if size == nil {
		return 0
	}

	return int(size.Int64())
}

func SubnetMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("subnets", MongoCollectionSubnetName)
	if err != nil {
		return err
	}

	for _, subnet := range FindSubnets() {
		if subnet.SchemaVersion == 0 || subnet.SchemaVersion == 1 {
			err := subnet.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewSubnet(vnetId, name, desc, cidr, gw, dns string, private bool, reserved, gwPool []sdk.IpRange, routes []string, app string) (*Subnet, error) {

	// ensure the requested subnet is correctly flagged
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if private && !ip.IsPrivate() {
		err := fmt.Errorf("trying to create a public subnet (%s) in a private virtual network", cidr)
		klog.Error(err)
		return nil, err
	}
	if !private && ip.IsPrivate() {
		err := fmt.Errorf("trying to create a private subnet (%s) in a public virtual network", cidr)
		klog.Error(err)
		return nil, err
	}

	s := Subnet{
		Resource:    NewResource(name, desc, MongoCollectionSubnetSchemaVersion),
		VNetID:      vnetId,
		CIDR:        cidr,
		Gateway:     gw,
		DNS:         dns,
		Reserved:    []*IPRange{},
		GwPool:      []*IPRange{},
		Routes:      routes,
		Application: app,
		AdapterIDs:  []string{},
	}
	if dns == "" {
		s.DNS = gw
	}

	// IPv4 ranges not to be used by Kowabunga
	for _, r := range reserved {
		if r.First != "" && r.Last != "" {
			if !s.IsValid(r.First) || !s.IsValid(r.Last) {
				err := fmt.Errorf("reserved IPv4 addresses %s are not part of %s CIDR", r, cidr)
				klog.Error(err)
				return nil, err
			}
			ipr := IPRange{r.First, r.Last}
			s.Reserved = append(s.Reserved, &ipr)
		}
	}

	// IPv4 ranges Kowabunga can use for Kawaii virtual IPs
	for _, g := range gwPool {
		if g.First != "" && g.Last != "" {
			if !s.IsValid(g.First) || !s.IsValid(g.Last) {
				err := fmt.Errorf("gateway pool IPv4 addresses %s are not part of %s CIDR", g, cidr)
				klog.Error(err)
				return nil, err
			}
			ipr := IPRange{g.First, g.Last}
			s.GwPool = append(s.GwPool, &ipr)
		}
	}
	err = s.IsValidGwPool()
	if err != nil {
		return nil, err
	}

	v, err := s.VNet()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionSubnetName, s)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new subnet %s", s.String())

	// add subnet to virtual network
	v.AddSubnet(s.String())

	return &s, nil
}

func ReservePrivateSubnet(regionId, projectId string, subnetSize int) (string, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return "", err
	}

	// start by looking for an available subnet at the requested size
	size := subnetSize
	for size > SubnetMaximumNetMaskSize {
		for _, vid := range r.VNets() {
			v, err := r.VNet(vid)
			if err != nil {
				continue
			}

			// ensure we're private
			if !v.Private {
				continue
			}

			for _, sid := range v.Subnets() {
				s, err := v.Subnet(sid)
				if err != nil {
					continue
				}

				// ensure project is not already assigned to some existing project
				if s.ProjectID != "" {
					continue
				}

				// parse CIDR
				p, err := netip.ParsePrefix(s.CIDR)
				if err != nil {
					continue
				}

				// ensure subnet's netmask is the requested one
				if p.Bits() != size {
					continue
				}

				// DONE: link project and subnet
				s.SetProject(projectId)
				return s.String(), nil
			}
		}

		// no such subnet available, trying to find one with a larger mask
		size -= 1
	}
	return "", fmt.Errorf("unable to assign a private /%d VPC subnet. None available", subnetSize)
}

func FindSubnets() []Subnet {
	return FindResources[Subnet](MongoCollectionSubnetName)
}

func FindSubnetsByProject(projectId string) ([]Subnet, error) {
	return FindResourcesByKey[Subnet](MongoCollectionSubnetName, "project_id", projectId)
}

func FindSubnetsByVNet(vnetId string) ([]Subnet, error) {
	return FindResourcesByKey[Subnet](MongoCollectionSubnetName, "vnet_id", vnetId)
}

func FindSubnetByID(id string) (*Subnet, error) {
	return FindResourceByID[Subnet](MongoCollectionSubnetName, id)
}

func FindSubnetByName(name string) (*Subnet, error) {
	return FindResourceByName[Subnet](MongoCollectionSubnetName, name)
}

func (s *Subnet) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionSubnetName, s.ID, from, to)
}

func (s *Subnet) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionSubnetName, s.ID, version)
}

func (s *Subnet) migrateSchemaV2() error {
	err := s.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = s.renameDbField("vnet", "vnet_id")
	if err != nil {
		return err
	}

	err = s.renameDbField("adapters", "adapter_ids")
	if err != nil {
		return err
	}

	err = s.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (s *Subnet) Project() (*Project, error) {
	return FindProjectByID(s.ProjectID)
}

func (s *Subnet) VNet() (*VNet, error) {
	return FindVNetByID(s.VNetID)
}

func (s *Subnet) HasChildren() bool {
	return HasChildRefs(s.AdapterIDs)
}

func (s *Subnet) FindAdapters() ([]Adapter, error) {
	return FindAdaptersBySubnet(s.String())
}

func (s *Subnet) SetProject(id string) {
	s.ProjectID = id
	s.Save()
}

func (s *Subnet) Size() int {
	p, err := netip.ParsePrefix(s.CIDR)
	if err != nil {
		return 0
	}

	return p.Bits()
}

func (s *Subnet) FindIPs() []string {
	ips := []string{}

	adapters, err := s.FindAdapters()
	if err != nil {
		return ips
	}

	for _, a := range adapters {
		ips = append(ips, a.Addresses...)
	}

	return ips
}

func (s *Subnet) IsValid(ip string) bool {
	network, err := netip.ParsePrefix(s.CIDR)
	if err != nil {
		return false
	}

	_ip, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	return network.Contains(_ip)
}

func (s *Subnet) IsInReservedPool(ip string) bool {
	ipaddr := net.ParseIP(ip)
	for _, r := range s.Reserved {
		first := net.ParseIP(r.First)
		last := net.ParseIP(r.Last)
		if bytes.Compare(ipaddr, first) >= 0 && bytes.Compare(ipaddr, last) <= 0 {
			return true
		}
	}
	return false
}

func (s *Subnet) IsInGwPool(ip string) bool {
	ipaddr := net.ParseIP(ip)
	for _, r := range s.GwPool {
		first := net.ParseIP(r.First)
		last := net.ParseIP(r.Last)
		if bytes.Compare(ipaddr, first) >= 0 && bytes.Compare(ipaddr, last) <= 0 {
			return true
		}
	}
	return false
}

func (s *Subnet) IsValidGwPool() error {
	for _, p := range s.GwPool {
		poolSize := p.Size()

		v, err := s.VNet()
		if err != nil {
			return err
		}

		r, err := v.Region()
		if err != nil {
			return err
		}

		if poolSize < len(r.Zones()) {
			err := fmt.Errorf("reserved gateway pool size (%d) is smaller than region's zones count (%d)", poolSize, len(r.Zones()))
			klog.Error(err)
			return err
		}
	}

	return nil
}

func (s *Subnet) FindGwPoolIPs() []string {
	ips := []string{}

	for _, p := range s.GwPool {
		first := ipa.NewIPAddressString(p.First).GetAddress()
		if first == nil {
			return ips
		}

		last := ipa.NewIPAddressString(p.Last).GetAddress()
		if last == nil {
			return ips
		}

		rg := ipa.NewIPSeqRange(first, last)
		if rg == nil {
			return ips
		}

		it := rg.Iterator()
		for addr := it.Next(); addr != nil; addr = it.Next() {
			ips = append(ips, addr.String())
		}
	}

	return ips
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (s *Subnet) FreeIPsCount() int {
	ip, ipnet, err := net.ParseCIDR(s.CIDR)
	if err != nil {
		return 0
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// sum of CIDR elligible addresses, minus network and broadcast addresses
	count := len(ips[1 : len(ips)-1])

	// minus reserved ranges
	for _, r := range s.Reserved {
		count -= r.Size()
	}

	// minus gateway pools
	for _, p := range s.GwPool {
		count -= p.Size()
	}

	// minus already used IP addresses
	count -= len(s.FindIPs())

	return count
}

func (s *Subnet) Update(name, desc, gw, dns string, reserved, gwPool []sdk.IpRange, routes []string, app string) error {
	s.UpdateResourceDefaults(name, desc)
	SetFieldStr(&s.Gateway, gw)
	SetFieldStr(&s.DNS, dns)
	reservedRanges := []*IPRange{}
	for _, r := range reserved {
		if r.First != "" && r.Last != "" {
			if !s.IsValid(r.First) || !s.IsValid(r.Last) {
				err := fmt.Errorf("reserved IPv4 addresses %s are not part of %s CIDR", r, s.CIDR)
				klog.Error(err)
				return err
			}
			ipr := IPRange{r.First, r.Last}
			reservedRanges = append(reservedRanges, &ipr)
		}
	}
	s.Reserved = reservedRanges
	gwPoolRanges := []*IPRange{}
	for _, g := range gwPool {
		if g.First != "" && g.Last != "" {
			if !s.IsValid(g.First) || !s.IsValid(g.Last) {
				err := fmt.Errorf("gateway pool IPv4 addresses %s are not part of %s CIDR", g, s.CIDR)
				klog.Error(err)
				return err
			}
			ipr := IPRange{g.First, g.Last}
			gwPoolRanges = append(gwPoolRanges, &ipr)
		}
	}
	s.GwPool = gwPoolRanges
	err := s.IsValidGwPool()
	if err != nil {
		return err
	}

	s.Routes = routes
	s.Application = app
	// we forbid change of CIDR or privacy, makes no sense
	s.Save()

	return nil
}

func (s *Subnet) Save() {
	s.Updated()
	_, err := GetDB().Update(MongoCollectionSubnetName, s.ID, s)
	if err != nil {
		klog.Error(err)
	}
}

func (s *Subnet) Delete() error {
	klog.Debugf("Deleting subnet %s", s.String())

	if s.String() == ResourceUnknown {
		return nil
	}

	// remove subnet's reference from parents
	v, err := s.VNet()
	if err != nil {
		return err
	}
	v.RemoveSubnet(s.String())

	return GetDB().Delete(MongoCollectionSubnetName, s.ID)
}

func (s *Subnet) Model() sdk.Subnet {
	reservedRanges := []sdk.IpRange{}
	for _, r := range s.Reserved {
		m := sdk.IpRange{
			First: r.First,
			Last:  r.Last,
		}
		reservedRanges = append(reservedRanges, m)
	}
	gwPoolRanges := []sdk.IpRange{}
	for _, g := range s.GwPool {
		m := sdk.IpRange{
			First: g.First,
			Last:  g.Last,
		}
		gwPoolRanges = append(gwPoolRanges, m)
	}
	return sdk.Subnet{
		Id:          s.String(),
		Name:        s.Name,
		Description: s.Description,
		Cidr:        s.CIDR,
		Gateway:     s.Gateway,
		Dns:         s.DNS,
		ExtraRoutes: s.Routes,
		Reserved:    reservedRanges,
		GwPool:      gwPoolRanges,
		Application: s.Application,
	}
}

// Adapters
func (s *Subnet) Adapters() []string {
	return s.AdapterIDs
}

func (s *Subnet) Adapter(id string) (*Adapter, error) {
	return FindChildByID[Adapter](&s.AdapterIDs, id, MongoCollectionAdapterName, ErrSubnetNoSuchAdapter)
}

func (s *Subnet) AddAdapter(id string) {
	klog.Debugf("Adding adapter %s to subnet %s", id, s.String())
	AddChildRef(&s.AdapterIDs, id)
	s.Save()
}

func (s *Subnet) RemoveAdapter(id string) {
	klog.Debugf("Removing adapter %s from subnet %s", id, s.String())
	RemoveChildRef(&s.AdapterIDs, id)
	s.Save()
}
