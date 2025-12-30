/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionVNetSchemaVersion = 2
	MongoCollectionVNetName          = "vnet"

	ErrVNetNoSuchSubnet = "no such subnet in virtual network"
)

type VNet struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	RegionID string `bson:"region_id"`

	// properties
	VLAN      int          `bson:"vlan_id"`
	Interface string       `bson:"interface"`
	Private   bool         `bson:"private"`
	Defaults  VNetDefaults `bson:"defaults"`

	// children references
	SubnetIDs []string `bson:"subnet_ids"`
}

type VNetDefaults struct {
	SubnetID string `bson:"subnet_id"`
}

func VNetMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("vnets", MongoCollectionVNetName)
	if err != nil {
		return err
	}

	for _, vnet := range FindVNets() {
		if vnet.SchemaVersion == 0 || vnet.SchemaVersion == 1 {
			err := vnet.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewVNet(regionId, name, desc string, vlan int, itf string, private bool) (*VNet, error) {
	v := VNet{
		Resource:  NewResource(name, desc, MongoCollectionVNetSchemaVersion),
		RegionID:  regionId,
		VLAN:      vlan,
		Interface: itf,
		Private:   private,
		SubnetIDs: []string{},
	}

	r, err := v.Region()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionVNetName, v)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new virtual network %s", v.String())

	// add vnet to region
	r.AddVNet(v.String())

	return &v, nil
}

func FindVNets() []VNet {
	return FindResources[VNet](MongoCollectionVNetName)
}

func FindVNetsByRegion(regionId string) ([]VNet, error) {
	return FindResourcesByKey[VNet](MongoCollectionVNetName, "region_id", regionId)
}

func FindVNetByID(id string) (*VNet, error) {
	return FindResourceByID[VNet](MongoCollectionVNetName, id)
}

func FindVNetByName(name string) (*VNet, error) {
	return FindResourceByName[VNet](MongoCollectionVNetName, name)
}

func (v *VNet) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionVNetName, v.ID, from, to)
}

func (v *VNet) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionVNetName, v.ID, version)
}

func (v *VNet) migrateSchemaV2() error {
	err := v.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = v.renameDbField("defaults.subnet", "defaults.subnet_id")
	if err != nil {
		return err
	}

	err = v.renameDbField("subnets", "subnet_ids")
	if err != nil {
		return err
	}

	err = v.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (v *VNet) Region() (*Region, error) {
	return FindRegionByID(v.RegionID)
}

func (v *VNet) HasChildren() bool {
	return HasChildRefs(v.SubnetIDs)
}

func (v *VNet) FindSubnets() ([]Subnet, error) {
	return FindSubnetsByVNet(v.String())
}

func (v *VNet) Update(name, desc string, vlan int, itf string) {
	v.UpdateResourceDefaults(name, desc)
	v.VLAN = vlan
	SetFieldStr(&v.Interface, itf)
	// we forbid change on privacy
	v.Save()
}

func (v *VNet) Save() {
	v.Updated()
	_, err := GetDB().Update(MongoCollectionVNetName, v.ID, v)
	if err != nil {
		klog.Error(err)
	}
}

func (v *VNet) Delete() error {
	klog.Debugf("Deleting virtual network %s", v.String())

	if v.String() == ResourceUnknown {
		return nil
	}

	// remove vnet's reference from parents
	r, err := v.Region()
	if err != nil {
		return err
	}
	r.RemoveVNet(v.String())

	return GetDB().Delete(MongoCollectionVNetName, v.ID)
}

func (v *VNet) Model() sdk.VNet {
	vlan := int64(v.VLAN)
	return sdk.VNet{
		Id:          v.String(),
		Name:        v.Name,
		Description: v.Description,
		Vlan:        vlan,
		Interface:   v.Interface,
		Private:     v.Private,
	}
}

// Subnets
func (v *VNet) Subnets() []string {
	return v.SubnetIDs
}

func (v *VNet) Subnet(id string) (*Subnet, error) {
	return FindChildByID[Subnet](&v.SubnetIDs, id, MongoCollectionSubnetName, ErrVNetNoSuchSubnet)
}

func (v *VNet) AddSubnet(id string) {
	klog.Debugf("Adding subnet %s to virtual network %s", id, v.String())
	AddChildRef(&v.SubnetIDs, id)
	// set subnet as default one if none exists
	err := v.SetDefaultSubnet(id, false)
	if err != nil {
		klog.Error(err)
	}
	v.Save()
}

func (v *VNet) RemoveSubnet(id string) {
	klog.Debugf("Removing subnet %s from virtual network %s", id, v.String())
	RemoveChildRef(&v.SubnetIDs, id)
	// possibly unset default pool
	if v.Defaults.SubnetID == id {
		v.Defaults.SubnetID = ""
	}
	v.Save()
}

func (v *VNet) SetDefaultSubnet(subnetId string, force bool) error {

	s, err := FindSubnetByID(subnetId)
	if err != nil {
		return err
	}

	if force || v.Defaults.SubnetID == "" {
		v.Defaults.SubnetID = s.String()
	}
	v.Save()

	return nil
}

func (v *VNet) FindFreeSubnet(requestedIps int) (*Subnet, error) {
	for _, subnetId := range v.SubnetIDs {
		s, err := FindSubnetByID(subnetId)
		if err != nil {
			return nil, err
		}

		if s.FreeIPsCount() >= requestedIps {
			return s, nil
		}
	}

	return nil, fmt.Errorf("no subnet with %d free IP addresses", requestedIps)
}
