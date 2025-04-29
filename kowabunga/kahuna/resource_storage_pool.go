/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kaktus"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionStoragePoolSchemaVersion = 2
	MongoCollectionStoragePoolName          = "storage_pool"

	StoragePoolCephMonitorPort = 3300

	ErrStoragePoolNoSuchVolume   = "no such volume in pool"
	ErrStoragePoolNoSuchTemplate = "no such template in pool"
)

type StoragePool struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	RegionID string `bson:"region_id"`

	// properties
	Pool       string              `bson:"libvirt_pool"`
	Address    string              `bson:"ceph_mon_address"`
	Port       int                 `bson:"ceph_mon_port"`
	Auth       string              `bson:"ceph_auth_secret_uuid"`
	Cost       ResourceCost        `bson:"cost"`
	Capacity   uint64              `bson:"capacity"`
	Allocation uint64              `bson:"allocation"`
	Available  uint64              `bson:"available"`
	Defaults   StoragePoolDefaults `bson:"defaults"`
	AgentIDs   []string            `bson:"agent_ids"`

	// children references
	TemplateIDs []string `bson:"template_ids"`
	VolumeIDs   []string `bson:"volume_ids"`
}

type StoragePoolDefaults struct {
	TemplateIDs StoragePoolDefaultsTemplates `bson:"template_ids"`
}

type StoragePoolDefaultsTemplates struct {
	OS string `bson:"os"`
}

func StoragePoolMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("pools", MongoCollectionStoragePoolName)
	if err != nil {
		return err
	}

	for _, pool := range FindStoragePools() {
		if pool.SchemaVersion == 0 || pool.SchemaVersion == 1 {
			err := pool.migrateSchemaV2()
			if err != nil {
				return err
			}

			poolReloaded, err := FindStoragePoolByID(pool.String())
			if err != nil {
				return err
			}

			var agents []string

			// KSA agents have been deprecated and merged into Kaktus ones, so use these instead
			region, err := poolReloaded.Region()
			if err != nil {
				return err
			}

			zones, err := region.FindZones()
			if err != nil {
				return err
			}

			for _, zone := range zones {
				for _, kaktusId := range zone.Kaktuses() {
					kaktus, err := FindKaktusByID(kaktusId)
					if err != nil {
						return err
					}

					agents = append(agents, kaktus.Agents()...)
				}
			}

			poolReloaded.AgentIDs = agents
			poolReloaded.Save()
		}
	}

	return nil
}

func NewStoragePool(regionId, name, desc, pool, address string, port int, secret string, price float32, currency string, agts []string) (*StoragePool, error) {
	p := StoragePool{
		Resource:    NewResource(name, desc, MongoCollectionStoragePoolSchemaVersion),
		RegionID:    regionId,
		Pool:        pool,
		Address:     address,
		Port:        port,
		Auth:        secret,
		Cost:        NewResourceCost(price, currency),
		AgentIDs:    VerifyAgents(agts, common.KowabungaKaktusAgent),
		TemplateIDs: []string{},
		VolumeIDs:   []string{},
	}

	if p.Port == 0 {
		p.Port = StoragePoolCephMonitorPort
	}

	r, err := p.Region()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionStoragePoolName, p)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new storage pool %s", p.String())

	// add storage pool to region
	r.AddStoragePool(p.String())

	return &p, nil
}

func FindStoragePools() []StoragePool {
	return FindResources[StoragePool](MongoCollectionStoragePoolName)
}

func FindStoragePoolsByRegion(regionId string) ([]StoragePool, error) {
	return FindResourcesByKey[StoragePool](MongoCollectionStoragePoolName, "region_id", regionId)
}

func FindStoragePoolByID(id string) (*StoragePool, error) {
	return FindResourceByID[StoragePool](MongoCollectionStoragePoolName, id)
}

func FindStoragePoolByName(name string) (*StoragePool, error) {
	return FindResourceByName[StoragePool](MongoCollectionStoragePoolName, name)
}

func (p *StoragePool) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionStoragePoolName, p.ID, from, to)
}

func (p *StoragePool) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionStoragePoolName, p.ID, version)
}

func (p *StoragePool) migrateSchemaV2() error {
	err := p.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = p.renameDbField("defaults.templates", "defaults.template_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("agents", "agent_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("templates", "template_ids")
	if err != nil {
		return err
	}

	err = p.renameDbField("volumes", "volume_ids")
	if err != nil {
		return err
	}

	err = p.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (p *StoragePool) RPC(method string, args, reply any) error {
	return RPC(p.AgentIDs, method, args, reply)
}

func (p *StoragePool) Region() (*Region, error) {
	return FindRegionByID(p.RegionID)
}

func (p *StoragePool) Agents() []string {
	return p.AgentIDs
}

func (p *StoragePool) HasChildren() bool {
	return HasChildRefs(p.TemplateIDs, p.VolumeIDs)
}

func (p *StoragePool) FindTemplates() ([]Template, error) {
	return FindTemplatesByStoragePool(p.String())
}

func (p *StoragePool) FindVolumes() ([]Volume, error) {
	return FindVolumesByStoragePool(p.String())
}

func (p *StoragePool) AverageRegionResources() (*RegionVirtualResources, error) {
	r, err := p.Region()
	if err != nil {
		return nil, err
	}

	return r.AverageVirtualResources(), nil
}

func (p *StoragePool) Scan() {
	klog.Debugf("Scanning Storage Pool %s", p)
	updated := false

	args := kaktus.KaktusGetStoragePoolStatsArgs{
		Pool: p.Pool,
	}
	var reply kaktus.KaktusGetStoragePoolStatsReply

	err := p.RPC("GetStoragePoolStats", args, &reply)
	if err != nil {
		klog.Errorf("Unable to get remote storage pool statistics: %v", err)
		return
	}

	p.Allocation = reply.Allocated
	p.Available = reply.Available
	if p.Capacity != reply.Capacity {
		p.Capacity = reply.Capacity
		updated = true
	}

	klog.Debugf("Storage pool %s (%s used / %s)", p.String(), byteCountIEC(reply.Allocated), byteCountIEC(reply.Capacity))

	p.Save()

	// if pool settings have changed, trigger a region capability update
	if updated {
		r, err := p.Region()
		if err != nil {
			return
		}

		go func() {
			err := r.UpdateCapabilities()
			if err != nil {
				klog.Error(err.Error())
			}
		}()
	}
}

func (p *StoragePool) Update(name, desc, pool, address string, port int, secret string, price float32, currency string, agts []string) {
	p.UpdateResourceDefaults(name, desc)

	SetFieldStr(&p.Pool, pool)
	SetFieldStr(&p.Address, address)
	p.Port = port
	if p.Port == 0 {
		p.Port = StoragePoolCephMonitorPort
	}
	SetFieldStr(&p.Auth, secret)
	p.Cost.Price = price
	SetFieldStr(&p.Cost.Currency, currency)
	p.AgentIDs = VerifyAgents(agts, common.KowabungaKaktusAgent)

	p.Save()
}

func (p *StoragePool) Save() {
	p.Updated()
	_, err := GetDB().Update(MongoCollectionStoragePoolName, p.ID, p)
	if err != nil {
		klog.Error(err)
	}
}

func (p *StoragePool) Delete() error {
	klog.Debugf("Deleting storage pool %s", p.String())

	if p.String() == ResourceUnknown {
		return nil
	}

	// remove region's reference from parents
	r, err := p.Region()
	if err != nil {
		return err
	}
	r.RemoveStoragePool(p.String())

	return GetDB().Delete(MongoCollectionStoragePoolName, p.ID)
}

func (p *StoragePool) Model() sdk.StoragePool {
	port := int64(p.Port)
	cost := p.Cost.Model()
	return sdk.StoragePool{
		Id:             p.String(),
		Name:           p.Name,
		Description:    p.Description,
		Pool:           p.Pool,
		CephAddress:    p.Address,
		CephPort:       port,
		CephSecretUuid: p.Auth,
		Cost:           cost,
		Agents:         p.AgentIDs,
	}
}

// Templates
func (p *StoragePool) Templates() []string {
	return p.TemplateIDs
}

func (p *StoragePool) Template(id string) (*Template, error) {
	return FindChildByID[Template](&p.TemplateIDs, id, MongoCollectionTemplateName, ErrStoragePoolNoSuchTemplate)
}

func (p *StoragePool) AddTemplate(id string) {
	klog.Debugf("Adding template %s to pool %s", id, p.String())
	AddChildRef(&p.TemplateIDs, id)
	// set template as default one if none exists
	err := p.SetDefaultTemplate(id, false)
	if err != nil {
		klog.Error(err)
	}
	p.Save()
}

func (p *StoragePool) RemoveTemplate(id string) {
	klog.Debugf("Removing template %s from pool %s", id, p.String())
	RemoveChildRef(&p.TemplateIDs, id)
	// possibly unset default template
	if p.Defaults.TemplateIDs.OS == id {
		p.Defaults.TemplateIDs.OS = ""
	}
	p.Save()
}

func (p *StoragePool) SetDefaultTemplate(id string, force bool) error {

	t, err := FindTemplateByID(id)
	if err != nil {
		return err
	}

	if force || p.Defaults.TemplateIDs.OS == "" {
		p.Defaults.TemplateIDs.OS = t.String()
	}
	p.Save()

	return nil
}

// Volumes
func (p *StoragePool) Volumes() []string {
	return p.VolumeIDs
}

func (p *StoragePool) Volume(id string) (*Volume, error) {
	return FindChildByID[Volume](&p.VolumeIDs, id, MongoCollectionVolumeName, ErrStoragePoolNoSuchVolume)
}

func (p *StoragePool) AddVolume(id string) {
	klog.Debugf("Adding volume %s to pool %s", id, p.String())
	AddChildRef(&p.VolumeIDs, id)
	p.Save()
}

func (p *StoragePool) RemoveVolume(id string) {
	klog.Debugf("Removing volume %s from pool %s", id, p.String())
	RemoveChildRef(&p.VolumeIDs, id)
	p.Save()
}
