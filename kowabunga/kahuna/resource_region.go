/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionRegionSchemaVersion = 2
	MongoCollectionRegionName          = "region"

	ErrRegionNoSuchZone        = "no such zone in region"
	ErrRegionNoSuchKiwi        = "no such network gateway in region"
	ErrRegionNoSuchVNet        = "no such virtual network in region"
	ErrRegionNoSuchStoragePool = "no such storage pool in region"
	ErrRegionNoSuchNfs         = "no such NFS storage in region"

	RegionMaxScore = 999999
)

type Region struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents

	// properties
	Defaults         RegionDefaults         `bson:"defaults"`
	VirtualResources RegionVirtualResources `bson:"virtual_resources"`

	// children references
	ZoneIDs        []string `bson:"zone_ids"`
	KiwiIDs        []string `bson:"kiwi_ids"`
	VNetIDs        []string `bson:"vnet_ids"`
	StoragePoolIDs []string `bson:"storage_pool_ids"`
	NfsIDs         []string `bson:"nfs_ids"`
}

type RegionDefaults struct {
	StoragePoolID string `bson:"pool_id"`
	NfsID         string `bson:"nfs_id"`
}

type RegionVirtualResources struct {
	Storage RegionVirtualResource `bson:"storage_gb"`
}

type RegionVirtualResource struct {
	Count    int64   `bson:"count"`
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func RegionMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("regions", MongoCollectionRegionName)
	if err != nil {
		return err
	}

	for _, region := range FindRegions() {
		if region.SchemaVersion == 0 || region.SchemaVersion == 1 {
			err := region.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewRegion(name, desc string) (*Region, error) {
	r := Region{
		Resource:       NewResource(name, desc, MongoCollectionRegionSchemaVersion),
		ZoneIDs:        []string{},
		KiwiIDs:        []string{},
		VNetIDs:        []string{},
		StoragePoolIDs: []string{},
		NfsIDs:         []string{},
	}

	_, err := GetDB().Insert(MongoCollectionRegionName, r)
	if err != nil {
		return nil, err
	}
	klog.Debugf("Created new region %s", r.String())

	return &r, nil
}

func FindRegions() []Region {
	return FindResources[Region](MongoCollectionRegionName)
}

func FindRegionByID(id string) (*Region, error) {
	return FindResourceByID[Region](MongoCollectionRegionName, id)
}

func FindRegionByName(name string) (*Region, error) {
	return FindResourceByName[Region](MongoCollectionRegionName, name)
}

func (r *Region) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionRegionName, r.ID, from, to)
}

func (r *Region) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionRegionName, r.ID, version)
}

func (r *Region) migrateSchemaV2() error {
	err := r.renameDbField("defaults.pool", "defaults.pool_id")
	if err != nil {
		return err
	}

	err = r.renameDbField("defaults.nfs", "defaults.nfs_id")
	if err != nil {
		return err
	}

	err = r.renameDbField("zones", "zone_ids")
	if err != nil {
		return err
	}

	err = r.renameDbField("netgws", "kiwi_ids")
	if err != nil {
		return err
	}

	err = r.renameDbField("vnets", "vnet_ids")
	if err != nil {
		return err
	}

	err = r.renameDbField("pools", "storage_pool_ids")
	if err != nil {
		return err
	}

	err = r.renameDbField("nfs", "nfs_ids")
	if err != nil {
		return err
	}

	err = r.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (r *Region) HasChildren() bool {
	return HasChildRefs(r.ZoneIDs, r.KiwiIDs, r.VNetIDs, r.StoragePoolIDs, r.NfsIDs)
}

func (r *Region) FindZones() ([]Zone, error) {
	return FindZonesByRegion(r.String())
}

func (r *Region) FindKiwis() ([]Kiwi, error) {
	return FindKiwisByRegion(r.String())
}

func (r *Region) FindVNets() ([]VNet, error) {
	return FindVNetsByRegion(r.String())
}

func (r *Region) FindStoragePools() ([]StoragePool, error) {
	return FindStoragePoolsByRegion(r.String())
}

func (r *Region) FindNfs() ([]NFS, error) {
	return FindNfsByRegion(r.String())
}

func (r *Region) AverageVirtualResources() *RegionVirtualResources {
	return &r.VirtualResources
}

func (r *Region) Update(name, desc string) {
	r.UpdateResourceDefaults(name, desc)
	r.Save()
}

func (r *Region) Save() {
	r.Updated()
	_, err := GetDB().Update(MongoCollectionRegionName, r.ID, r)
	if err != nil {
		klog.Error(err)
	}
}

func (r *Region) Delete() error {
	klog.Debugf("Deleting region %s", r.String())

	if r.String() == ResourceUnknown {
		return nil
	}

	return GetDB().Delete(MongoCollectionRegionName, r.ID)
}

func (r *Region) Model() sdk.Region {
	return sdk.Region{
		Id:          r.String(),
		Name:        r.Name,
		Description: r.Description,
	}
}

// list of $count zones with the fewest computing load
func (r *Region) ElectMostFavorableZones(count int) ([]string, error) {
	var zones []string

	zoneScores := make(map[string]int)
	for _, zoneId := range r.ZoneIDs {
		z, err := FindZoneByID(zoneId)
		if err != nil {
			return zones, err
		}

		zoneScores[z.String()] = z.UsageScore()
	}

	for i := 0; i < count; i++ {
		bestScore := RegionMaxScore
		bestZone := ""

		for zoneId, score := range zoneScores {
			if score < bestScore {
				// we got a potential winner
				bestScore = score
				bestZone = zoneId
			}
		}
		if bestZone != "" {
			zones = append(zones, bestZone)
			delete(zoneScores, bestZone)
		}
	}

	return zones, nil
}

// Zones

func (r *Region) Zones() []string {
	return r.ZoneIDs
}

func (r *Region) Zone(id string) (*Zone, error) {
	return FindChildByID[Zone](&r.ZoneIDs, id, MongoCollectionZoneName, ErrRegionNoSuchZone)
}

func (r *Region) AddZone(id string) {
	klog.Debugf("Adding zone %s to region %s", id, r.String())
	AddChildRef(&r.ZoneIDs, id)
	r.Save()
}

func (r *Region) RemoveZone(id string) {
	klog.Debugf("Removing zone %s from region %s", id, r.String())
	RemoveChildRef(&r.ZoneIDs, id)
	r.Save()
}

// Network Gateways

func (r *Region) Kiwis() []string {
	return r.KiwiIDs
}

func (r *Region) Kiwi(id string) (*Kiwi, error) {
	return FindChildByID[Kiwi](&r.KiwiIDs, id, MongoCollectionKiwiName, ErrRegionNoSuchKiwi)
}

func (r *Region) AddKiwi(id string) {
	klog.Debugf("Adding network gateway %s to region %s", id, r.String())
	AddChildRef(&r.KiwiIDs, id)
	r.Save()
}

func (r *Region) RemoveKiwi(id string) {
	klog.Debugf("Removing kiwi %s from region %s", id, r.String())
	RemoveChildRef(&r.KiwiIDs, id)
	r.Save()
}

// Virtual Networks

func (r *Region) VNets() []string {
	return r.VNetIDs
}

func (r *Region) VNet(id string) (*VNet, error) {
	return FindChildByID[VNet](&r.VNetIDs, id, MongoCollectionVNetName, ErrRegionNoSuchVNet)
}

func (r *Region) AddVNet(id string) {
	klog.Debugf("Adding virtual network %s to region %s", id, r.String())
	AddChildRef(&r.VNetIDs, id)
	r.Save()
}

func (r *Region) RemoveVNet(id string) {
	klog.Debugf("Removing virtual network %s from region %s", id, r.String())
	RemoveChildRef(&r.VNetIDs, id)
	r.Save()
}

func (r *Region) GetPublicSubnet() (string, error) {
	for _, vid := range r.VNetIDs {
		v, err := FindVNetByID(vid)
		if err != nil {
			return "", err
		}

		// ensure we're public
		if v.Private {
			continue
		}

		return v.Defaults.SubnetID, nil
	}

	return "", fmt.Errorf("unable to assign a public IPv4 addess. None available")
}

// Storage Pools

func (r *Region) StoragePools() []string {
	return r.StoragePoolIDs
}

func (r *Region) StoragePool(id string) (*StoragePool, error) {
	return FindChildByID[StoragePool](&r.StoragePoolIDs, id, MongoCollectionRegionName, ErrRegionNoSuchStoragePool)
}

func (r *Region) AddStoragePool(id string) {
	klog.Debugf("Adding storage pool %s to region %s", id, r.String())
	AddChildRef(&r.StoragePoolIDs, id)

	// set storage pool as default one if none exists
	err := r.SetDefaultStoragePool(id, false)
	if err != nil {
		klog.Error(err)
	}
	r.Save()
}

func (r *Region) RemoveStoragePool(id string) {
	klog.Debugf("Removing storage pool %s from region %s", id, r.String())
	RemoveChildRef(&r.StoragePoolIDs, id)
	// possibly unset default storage pool
	if r.Defaults.StoragePoolID == id {
		r.Defaults.StoragePoolID = ""
	}
	err := r.UpdateCapabilities()
	if err != nil {
		klog.Error(err.Error())
	}
	r.Save()
}

func (r *Region) SetDefaultStoragePool(id string, force bool) error {
	pool, err := FindStoragePoolByID(id)
	if err != nil {
		return err
	}

	if force || r.Defaults.StoragePoolID == "" {
		r.Defaults.StoragePoolID = pool.String()
	}
	r.Save()

	return nil
}

// NFS Storages

func (r *Region) Nfses() []string {
	return r.NfsIDs
}

func (r *Region) Nfs(id string) (*NFS, error) {
	return FindChildByID[NFS](&r.NfsIDs, id, MongoCollectionNfsName, ErrRegionNoSuchNfs)
}

func (r *Region) AddNfs(id string) {
	klog.Debugf("Adding NFS storage %s to region %s", id, r.String())
	AddChildRef(&r.NfsIDs, id)

	// set NFS storage as default one if none exists
	err := r.SetDefaultNfs(id, false)
	if err != nil {
		klog.Error(err)
	}
	r.Save()
}

func (r *Region) RemoveNfs(id string) {
	klog.Debugf("Removing NFS storage %s from region %s", id, r.String())
	RemoveChildRef(&r.NfsIDs, id)
	// possibly unset default pool
	if r.Defaults.NfsID == id {
		r.Defaults.NfsID = ""
	}
	r.Save()
}

func (r *Region) SetDefaultNfs(id string, force bool) error {
	n, err := FindNfsByID(id)
	if err != nil {
		return err
	}

	if force || r.Defaults.NfsID == "" {
		r.Defaults.NfsID = n.String()
	}
	r.Save()

	return nil
}

// Cost
func (r *Region) UpdateCapabilities() error {
	klog.Debugf("Updating region %s virtual resources capabilities", r)

	res := RegionVirtualResources{}

	rbdPools := 0
	for _, poolId := range r.StoragePoolIDs {
		p, err := FindStoragePoolByID(poolId)
		if err != nil {
			return err
		}

		rbdPools += 1
		gbs := bytesToGB(int64(p.Capacity))
		res.Storage.Count += gbs
		res.Storage.Price += (p.Cost.Price / float32(gbs))
		res.Storage.Currency = p.Cost.Currency
	}
	res.Storage.Price /= float32(rbdPools)

	klog.Debugf("Region %s vStorage GB count: %d", r, res.Storage.Count)
	klog.Debugf("Region %s vStorage GB average price: %f %s", r, res.Storage.Price, res.Storage.Currency)

	r.VirtualResources = res
	r.Save()

	// triggers cost recomputation of all of region's storage pools
	for _, poolId := range r.StoragePoolIDs {
		p, err := FindStoragePoolByID(poolId)
		if err != nil {
			return err
		}

		for _, volumeId := range p.Volumes() {
			v, err := FindVolumeByID(volumeId)
			if err != nil {
				return err
			}
			err = v.ComputeCost(&res)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
