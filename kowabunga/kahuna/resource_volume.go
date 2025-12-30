/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"os"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/proto"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionVolumeSchemaVersion = 2
	MongoCollectionVolumeName          = "volume"

	VolumeOsDiskPrefix  = "vd"
	VolumeCloudInitDisk = "hdd"
	VolumeTypeOs        = "os"
	VolumeTypeRaw       = "raw"
	VolumeTypeIso       = "iso"
	VolumeTypeTemplate  = "template"
)

type Volume struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID     string `bson:"project_id"`
	StoragePoolID string `bson:"storage_pool_id"`

	// properties
	TemplateID string     `bson:"template_id"`
	Type       string     `bson:"type"`
	Size       int64      `bson:"size"`
	Cost       VolumeCost `bson:"cost"`

	// children references
}

type VolumeCost struct {
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func VolumeMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("volumes", MongoCollectionVolumeName)
	if err != nil {
		return err
	}

	for _, volume := range FindVolumes() {
		if volume.SchemaVersion == 0 || volume.SchemaVersion == 1 {
			err := volume.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewVolume(projectId, poolId, templateId, name, desc, tp string, size int64) (*Volume, error) {

	switch tp {
	case VolumeTypeOs, VolumeTypeIso, VolumeTypeRaw, VolumeTypeTemplate:
		break
	default:
		return nil, fmt.Errorf("unsupported volume type: %s", tp)
	}

	v := Volume{
		Resource:      NewResource(name, desc, MongoCollectionVolumeSchemaVersion),
		ProjectID:     projectId,
		StoragePoolID: poolId,
		TemplateID:    templateId,
		Type:          tp,
		Size:          size,
		Cost:          VolumeCost{},
	}

	p, err := v.StoragePool()
	if err != nil {
		return nil, err
	}

	err = v.CreateVolume()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionVolumeName, v)
	if err != nil {
		return nil, err
	}

	klog.Infof("Created new volume %s (%s)", v.String(), v.Name)

	// setup initial cost
	arr, err := v.AverageRegionResources()
	if err != nil {
		return nil, err
	}

	err = v.ComputeCost(arr)
	if err != nil {
		return nil, err
	}

	if tp != VolumeTypeTemplate {
		prj, err := v.Project()
		if err != nil {
			return nil, err
		}

		// add volume to project
		prj.AddVolume(v.String())
	}

	// add volume to pool
	p.AddVolume(v.String())

	return &v, nil
}

func buildLocalCloudInitImage(projectId, zoneId, instanceId, agentId, instanceName, instanceIP, osType, password, profile, domain, user, pubkey string, adapters []string) (*CloudInit, error) {
	ci, err := NewCloudInit(instanceName, osType)
	if err != nil {
		return nil, err
	}

	err = ci.SetUserData(instanceName, domain, password, user, pubkey, profile, adapters)
	if err != nil {
		klog.Error(err)
		return ci, err
	}

	err = ci.SetMetaData(profile, zoneId, instanceId, agentId, instanceName, instanceIP)
	if err != nil {
		klog.Error(err)
		return ci, err
	}

	err = ci.SetNetworkConfig(projectId, zoneId, domain, profile, adapters)
	if err != nil {
		klog.Error(err)
		return ci, err
	}

	err = ci.WriteISO()
	if err != nil {
		return ci, err
	}

	return ci, nil
}

func NewCloudInitVolume(projectId, zoneId, poolId, instanceId, agentId, instanceName, instanceIP, osType, password, profile string, adapters []string) (*Volume, error) {
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	ci, err := buildLocalCloudInitImage(projectId, zoneId, instanceId, agentId, instanceName, instanceIP, osType,
		password, profile, prj.Domain, prj.BootstrapUser, prj.BootstrapPubkey, adapters)
	if err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("%s cloudinit bootstrap image", instanceName)
	klog.Infof("Creating new cloud-init ISO image for instance %s on pool %s", instanceName, poolId)
	v, err := NewVolume(projectId, poolId, ci.IsoImage, ci.Name, desc, VolumeTypeIso, ci.IsoSize)
	if err != nil {
		return v, err
	}

	err = ci.Delete()
	if err != nil {
		return v, err
	}

	return v, nil
}

// Updates the Cloud Init ISO of a defined instance ID
// This does not create a new volume
func UpdateCloudInitIso(instanceID string) error {

	// Get our useful resources
	i, err := FindInstanceByID(instanceID)
	if err != nil {
		return err
	}
	prj, err := FindProjectByID(i.ProjectID)
	if err != nil {
		return err
	}
	v, err := FindVolumeByID(i.CloudInitVolumeId)
	if err != nil {
		return err
	}
	k, err := i.Kaktus()
	if err != nil {
		return err
	}
	z, err := k.Zone()
	if err != nil {
		return err
	}

	// Create our new ISO based on our existing instances
	ci, err := buildLocalCloudInitImage(i.ProjectID, z.String(), i.String(), i.AgentID, i.Name, i.LocalIP, v.Type, i.RootPassword, i.Profile, prj.Domain, prj.BootstrapUser, prj.BootstrapPubkey, i.Adapters())
	if err != nil {
		return err
	}

	// Update Our volume ISO through upload
	err = v.OverwriteCloudInitVolume(ci)
	if err != nil {
		return err
	}
	return nil
}

func FindVolumes() []Volume {
	return FindResources[Volume](MongoCollectionVolumeName)
}

func FindVolumesByProject(projectId string) ([]Volume, error) {
	return FindResourcesByKey[Volume](MongoCollectionVolumeName, "project_id", projectId)
}

func FindVolumesByStoragePool(poolId string) ([]Volume, error) {
	return FindResourcesByKey[Volume](MongoCollectionVolumeName, "storage_pool_id", poolId)
}

func FindVolumeByID(id string) (*Volume, error) {
	return FindResourceByID[Volume](MongoCollectionVolumeName, id)
}

func FindVolumeByName(name string) (*Volume, error) {
	return FindResourceByName[Volume](MongoCollectionVolumeName, name)
}

func (v *Volume) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionVolumeName, v.ID, from, to)
}

func (v *Volume) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionVolumeName, v.ID, version)
}

func (v *Volume) migrateSchemaV2() error {
	err := v.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = v.renameDbField("pool", "storage_pool_id")
	if err != nil {
		return err
	}

	err = v.renameDbField("template", "template_id")
	if err != nil {
		return err
	}

	err = v.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (v *Volume) Project() (*Project, error) {
	return FindProjectByID(v.ProjectID)
}

func (v *Volume) StoragePool() (*StoragePool, error) {
	return FindStoragePoolByID(v.StoragePoolID)
}

func (v *Volume) Template() (*Template, error) {
	return FindTemplateByID(v.TemplateID)
}

func (v *Volume) AverageRegionResources() (*RegionVirtualResources, error) {
	p, err := v.StoragePool()
	if err != nil {
		return nil, err
	}

	return p.AverageRegionResources()
}

func (v *Volume) ComputeCost(res *RegionVirtualResources) error {
	storage_gb := float64(bytesToGB(v.Size))
	currency := res.Storage.Currency
	price := float32(storage_gb) * res.Storage.Price
	klog.Debugf("Volume %s features %f GB for %f %s", v, storage_gb, price, currency)

	v.Cost.Price = price
	v.Cost.Currency = currency
	v.Save()

	return nil
}

func (v *Volume) Exists() bool {
	pool, err := v.StoragePool()
	if err != nil {
		return false
	}

	// get current volume infos
	args := proto.KaktusGetVolumeInfosArgs{
		Pool:   pool.Pool,
		Volume: v.Name,
	}
	var reply proto.KaktusGetVolumeInfosReply

	err = pool.RPC(proto.RpcKaktusGetVolumeInfos, args, &reply)
	if err != nil {
		klog.Errorf("Unable to get remote RBD volume: %v", err)
		return false
	}

	return true
}

func (v *Volume) ResizeVolume() error {

	pool, err := v.StoragePool()
	if err != nil {
		return err
	}

	// get current volume infos
	args := proto.KaktusGetVolumeInfosArgs{
		Pool:   pool.Pool,
		Volume: v.Name,
	}
	var reply proto.KaktusGetVolumeInfosReply

	err = pool.RPC(proto.RpcKaktusGetVolumeInfos, args, &reply)
	if err != nil {
		klog.Errorf("Unable to get remote RBD volume: %v", err)
		return err
	}
	capacity := reply.Size

	requestedSize := uint64(v.Size)
	if requestedSize < capacity {
		return fmt.Errorf("unable to shrink storage volume %s from %s to %s", v.String(), HumanByteSize(capacity), HumanByteSize(requestedSize))
	}

	// resize volume
	klog.Debugf("Resizing volume %s to %s", v.String(), HumanByteSize(requestedSize))
	argsResize := proto.KaktusResizeVolumeArgs{
		Pool:   pool.Pool,
		Volume: v.Name,
		Size:   requestedSize,
	}
	var replyResize proto.KaktusResizeVolumeReply

	err = pool.RPC(proto.RpcKaktusResizeVolume, argsResize, &replyResize)
	if err != nil {
		return fmt.Errorf("error trying to resize volume %s: %v", v.Name, err)
	}

	// update volume cost details
	arr, err := v.AverageRegionResources()
	if err != nil {
		return err
	}

	err = v.ComputeCost(arr)
	if err != nil {
		return err
	}

	return nil
}

func (v *Volume) RPC(method string, args, reply any) error {
	pool, err := v.StoragePool()
	if err != nil {
		return err
	}

	return pool.RPC(method, args, reply)
}

func (v *Volume) createRaw(pool, vol string, size int64) error {
	args := proto.KaktusCreateRawVolumeArgs{
		Pool:   pool,
		Volume: vol,
		Size:   uint64(size),
	}
	var reply proto.KaktusCreateRawVolumeReply

	return v.RPC(proto.RpcKaktusCreateRawVolume, args, &reply)
}

func (v *Volume) createTemplate(pool, vol, url string) (uint64, error) {
	args := proto.KaktusCreateTemplateVolumeArgs{
		Pool:      pool,
		Volume:    vol,
		SourceURL: url,
	}
	var reply proto.KaktusCreateTemplateVolumeReply

	err := v.RPC(proto.RpcKaktusCreateTemplateVolume, args, &reply)
	if err != nil {
		return 0, err
	}

	return reply.Size, nil
}

func (v *Volume) createOS(pool, vol, tpl string, size int64) error {
	args := proto.KaktusCreateOsVolumeArgs{
		Pool:     pool,
		Volume:   vol,
		Size:     uint64(size),
		Template: tpl,
	}
	var reply proto.KaktusCreateOsVolumeReply

	return v.RPC(proto.RpcKaktusCreateOsVolume, args, &reply)
}

func (v *Volume) createISO(pool, vol string, content []byte, size int64) error {
	args := proto.KaktusCreateIsoVolumeArgs{
		Pool:    pool,
		Volume:  vol,
		Size:    uint64(size),
		Content: content,
	}
	var reply proto.KaktusCreateIsoVolumeReply

	return v.RPC(proto.RpcKaktusCreateIsoVolume, args, &reply)
}

func (v *Volume) CreateVolume() error {

	pool, err := v.StoragePool()
	if err != nil {
		return err
	}

	switch v.Type {
	case VolumeTypeRaw:
		klog.Debugf("Creating RAW volume %s", v.String())
		err := v.createRaw(pool.Pool, v.Name, v.Size)
		if err != nil {
			klog.Errorf("Unable to create remote rws RBD volume: %v", err)
			return err
		}
	case VolumeTypeOs:
		t, err := v.Template()
		if err != nil {
			return err
		}

		klog.Debugf("Creating OS volume %s from template %s", v.String(), t.Name)
		err = v.createOS(pool.Pool, v.Name, t.Name, v.Size)
		if err != nil {
			klog.Errorf("Unable to create remote OS RBD volume: %v", err)
			return err
		}
	case VolumeTypeIso:
		// read up local ISO image
		imgContent, err := os.ReadFile(v.TemplateID)
		if err != nil {
			return err
		}

		klog.Debugf("Creating ISO volume %s", v.String())
		err = v.createISO(pool.Pool, v.Name, imgContent, v.Size)
		if err != nil {
			klog.Errorf("Unable to create remote ISO RBD volume: %v", err)
			return err
		}
	case VolumeTypeTemplate:
		// use templateId as source URL, download and create
		klog.Debugf("Creating template volume %s from template %s", v.Name, v.TemplateID)
		size, err := v.createTemplate(pool.Pool, v.Name, v.TemplateID)
		if err != nil {
			klog.Errorf("Unable to create remote template RBD volume: %v", err)
			return err
		}
		v.Size = int64(size)
	}

	return nil
}

func (v *Volume) Update(name, desc string, size int64) error {
	prj, err := v.Project()
	if err != nil {
		return err
	}

	v.UpdateResourceDefaults(name, desc)
	// one can't change volume type or resizable setting
	sizeDelta := size - v.Size
	if sizeDelta > 0 {
		v.Size = size
		err := v.ResizeVolume()
		if err != nil {
			return err
		}

		// update project usage counter
		prj.UpdateVolumeUsage(sizeDelta)
	}
	v.Save()
	return nil
}

func (v *Volume) Save() {
	v.Updated()
	_, err := GetDB().Update(MongoCollectionVolumeName, v.ID, v)
	if err != nil {
		klog.Error(err)
	}
}

func (v *Volume) Delete() error {
	klog.Infof("Deleting %s volume %s (%s)", v.Type, v.String(), v.Name)

	if v.String() == ResourceUnknown {
		return nil
	}

	pool, err := v.StoragePool()
	if err != nil {
		return err
	}
	args := proto.KaktusDeleteVolumeArgs{
		Pool:          pool.Pool,
		Volume:        v.Name,
		WithSnapshots: true,
	}
	var reply proto.KaktusDeleteVolumeReply

	err = pool.RPC(proto.RpcKaktusDeleteVolume, args, &reply)
	if err != nil {
		klog.Errorf("Unable to delete remote RBD volume: %v", err)
		return err
	}

	if v.Type != VolumeTypeTemplate {
		// remove volume's reference from parents
		prj, err := v.Project()
		if err != nil {
			return err
		}
		prj.RemoveVolume(v.String())
	}

	p, err := v.StoragePool()
	if err != nil {
		return err
	}
	p.RemoveVolume(v.String())

	return GetDB().Delete(MongoCollectionVolumeName, v.ID)
}

func (v *Volume) OverwriteCloudInitVolume(iso *CloudInit) error {

	pool, err := v.StoragePool()
	if err != nil {
		return err
	}

	// read up local ISO image
	imgContent, err := os.ReadFile(iso.IsoImage)
	if err != nil {
		return err
	}

	args := proto.KaktusUpdateIsoVolumeArgs{
		Pool:    pool.Pool,
		Volume:  v.Name,
		Size:    uint64(v.Size),
		Content: imgContent,
	}
	var reply proto.KaktusUpdateIsoVolumeReply

	klog.Debugf("Updating ISO content into volume %s", v.String())
	v.Size = iso.IsoSize
	err = pool.RPC(proto.RpcKaktusUpdateIsoVolume, args, &reply)
	if err != nil {
		klog.Errorf("Unable to create remote ISO RBD volume: %v", err)
		return err
	}

	return nil
}
func (v *Volume) Model() sdk.Volume {
	return sdk.Volume{
		Id:          v.String(),
		Name:        v.Name,
		Description: v.Description,
		Type:        v.Type,
		Size:        v.Size,
	}
}
