/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/netip"
	"slices"

	"github.com/seancfoley/ipaddress-go/ipaddr"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionAdapterSchemaVersion = 2
	MongoCollectionAdapterName          = "adapter"

	AdapterOsNicLinuxPrefix       = "ens"
	AdapterOsNicLinuxStartIndex   = 3
	AdapterOsNicWindowsPrefix     = "interface"
	AdapterOsNicWindowsStartIndex = 0
)

type Adapter struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	SubnetID string `bson:"subnet_id"`

	// properties
	MAC       string   `bson:"mac"`
	Addresses []string `bson:"addresses"`
	Reserved  bool     `bson:"reserved"`

	// children references
}

func AdapterMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("adapters", MongoCollectionAdapterName)
	if err != nil {
		return err
	}

	for _, adapter := range FindAdapters() {
		if adapter.SchemaVersion == 0 || adapter.SchemaVersion == 1 {
			err := adapter.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func RandomAdapterMAC() string {
	// QEMU vendor prefix - 52:54:00
	// 52:54:00:00:00:00 to 52:54:00:FF:FF:FF
	// Kowabunga vendor prefix - da:1e:70
	// da:1e:70:00:00:00 to da:1e:70:FF:FF:FF
	// This is still 2^24 (= 16 777 216) possibilities.
	for {
		var hw = make(net.HardwareAddr, 6)
		hw[0] = 0xda
		hw[1] = 0x1e
		hw[2] = 0x70

		// read is safe for concurrent use
		_, err := rand.Read(hw[3:])
		if err != nil {
			continue
		}

		mac := hw.String()

		// ensure we're not out of luck ...
		if slices.Contains(FindMACs(), mac) {
			continue
		}

		return mac
	}
}

func RandomIpAddress(subnetId string) (string, error) {
	var ip string

	s, err := FindSubnetByID(subnetId)
	if err != nil {
		return ip, err
	}

	p, err := netip.ParsePrefix(s.CIDR)
	if err != nil {
		return ip, err
	}
	p = p.Masked() // a.b.c.d/mask => a.b.c.0/mask

	// broadcast address
	cidr := ipaddr.NewIPAddressString(s.CIDR).GetAddress()
	bcast, _ := cidr.ToIPv4().ToBroadcastAddress()

	// find all subnet IPs
	registeredIPs := s.FindIPs()

	addr := p.Addr()
	addr = addr.Next() // skip the first IP from the range (.0)
	for p.Contains(addr) {
		ip = addr.String()

		// ensure it's not gateway or broadcast address
		if ip == s.Gateway || ip == bcast.String() {
			// loop over
			addr = addr.Next()
			continue
		}
		// ensure it's not part of the subnet's reserved pool ranges or zone-local gateway pool ranges
		if s.IsInReservedPool(ip) || s.IsInGwPool(ip) {
			// loop over
			addr = addr.Next()
			continue
		}
		// ensure it's not already assigned to other adapters in the subnet
		if slices.Contains(registeredIPs, ip) {
			// loop over
			addr = addr.Next()
			continue
		}

		// seems unused, go for it
		return ip, nil
	}

	return ip, fmt.Errorf("no IP can be assigned")
}

func verifyAdapterSettings(subnetId, mac string, addresses []string, update bool) error {
	s, err := FindSubnetByID(subnetId)
	if err != nil {
		return err
	}

	_, cidr, err := net.ParseCIDR(s.CIDR)
	if err != nil {
		return err
	}

	// ensure it's a correctly formatted IEEE 802 MAC-48 address
	_, err = net.ParseMAC(mac)
	if err != nil {
		return err
	}

	if !update {
		// ensure we don't have any MAC duplicates
		for _, a := range FindAdapters() {
			if mac == a.MAC && !a.Reserved {
				return fmt.Errorf("duplicated MAC address (%s) has been found. We can't authorize that unless you flag is as reserved", mac)
			}
		}
	}

	// ensure assigned IPv4 are in the associated subnet's CIDR range
	for _, i := range addresses {
		ip := net.ParseIP(i)
		if ip == nil {
			return fmt.Errorf("invalid IP: %s", i)
		}
		if !cidr.Contains(ip) {
			return fmt.Errorf("IP %s is not part of subnet's CIDR %s", i, cidr)
		}
	}

	return nil
}

func NewAdapter(subnetId, name, desc, mac string, addresses []string, reserved, autoAssign bool) (*Adapter, error) {
	if mac == "" {
		// no MAc address was provided, i.e. user requested one to be auto-generated
		mac = RandomAdapterMAC()
	}

	// randomly assign one IP address from the subnet if requested
	if autoAssign && len(addresses) == 0 {
		ip, err := RandomIpAddress(subnetId)
		if err != nil {
			return nil, err
		}
		klog.Debugf("IP address %s is being assigned.", ip)
		addresses = append(addresses, ip)
	}

	// ensure the requested adapter is correctly flagged
	err := verifyAdapterSettings(subnetId, mac, addresses, false)
	if err != nil {
		klog.Errorf("verifyAdapterSettings: %s", err)
		return nil, err
	}

	a := Adapter{
		Resource:  NewResource(name, desc, MongoCollectionAdapterSchemaVersion),
		SubnetID:  subnetId,
		MAC:       mac,
		Addresses: addresses,
		Reserved:  reserved,
	}

	s, err := a.Subnet()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionAdapterName, a)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new adapter %s", a.String())

	// add adapter to subnet
	s.AddAdapter(a.String())

	return &a, nil
}

func FindMACs() []string {
	macs := []string{}
	for _, a := range FindAdapters() {
		macs = append(macs, a.MAC)
	}
	return macs
}

func FindAdapters() []Adapter {
	return FindResources[Adapter](MongoCollectionAdapterName)
}

func FindAdaptersBySubnet(subnetId string) ([]Adapter, error) {
	return FindResourcesByKey[Adapter](MongoCollectionAdapterName, "subnet_id", subnetId)
}

func FindAdapterByID(id string) (*Adapter, error) {
	return FindResourceByID[Adapter](MongoCollectionAdapterName, id)
}

func FindAdapterByName(name string) (*Adapter, error) {
	return FindResourceByName[Adapter](MongoCollectionAdapterName, name)
}

func (a *Adapter) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionAdapterName, a.ID, from, to)
}

func (a *Adapter) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionAdapterName, a.ID, version)
}

func (a *Adapter) migrateSchemaV2() error {
	err := a.renameDbField("subnet", "subnet_id")
	if err != nil {
		return err
	}

	err = a.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (a *Adapter) Subnet() (*Subnet, error) {
	return FindSubnetByID(a.SubnetID)
}

func (a *Adapter) Update(name, desc, mac string, addresses []string, reserved bool) {
	// ensure the requested adapter is correctly flagged
	err := verifyAdapterSettings(a.SubnetID, mac, addresses, true)
	if err != nil {
		klog.Error(err)
		return
	}

	a.UpdateResourceDefaults(name, desc)
	SetFieldStr(&a.MAC, mac)
	a.Addresses = addresses
	a.Reserved = reserved
	a.Save()
}

func (a *Adapter) Save() {
	a.Updated()
	_, err := GetDB().Update(MongoCollectionAdapterName, a.ID, a)
	if err != nil {
		klog.Error(err)
	}
}

func (a *Adapter) Delete() error {
	klog.Debugf("Deleting adapter %s", a.String())

	if a.String() == ResourceUnknown {
		return nil
	}

	// remove zone's reference from parents
	s, err := a.Subnet()
	if err != nil {
		return err
	}
	s.RemoveAdapter(a.String())

	return GetDB().Delete(MongoCollectionAdapterName, a.ID)
}

func (a *Adapter) Model() sdk.Adapter {
	return sdk.Adapter{
		Id:          a.String(),
		Name:        a.Name,
		Description: a.Description,
		Mac:         a.MAC,
		Addresses:   a.Addresses,
		Reserved:    a.Reserved,
	}
}
