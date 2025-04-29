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
	MongoCollectionKomputeSchemaVersion = 2
	MongoCollectionKomputeName          = "kompute"
)

type Kompute struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`

	// properties
	InstanceID string `bson:"instance_id"`

	// children references
}

func KomputeMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("kces", MongoCollectionKomputeName)
	if err != nil {
		return err
	}

	for _, kompute := range FindKomputes() {
		if kompute.SchemaVersion == 0 || kompute.SchemaVersion == 1 {
			err := kompute.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewKompute(projectId, zoneId, hostId, poolId, templateId, name, desc, profile, profileId string, cpu, mem, disk, data int64, public bool, subnetPeerings []string) (*Kompute, error) {

	// ensure we have a rightful hostname, if any
	if !VerifyHostname(name) {
		err := fmt.Errorf("invalid host name: %s", name)
		klog.Error(err)
		return nil, err
	}

	// find parent objects, allows to bail before creating anything
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return nil, err
	}
	r, err := z.Region()
	if err != nil {
		return nil, err
	}

	privateSubnetId, err := prj.GetPrivateSubnet(r.String())
	if err != nil {
		return nil, err
	}

	// we do that at instance creation, not project creation because we might run out of free IPs in between
	publicSubnetId, err := r.GetPublicSubnet()
	if err != nil {
		return nil, err
	}

	volumes := []string{}

	// create an OS volume from template
	osVolumeName := fmt.Sprintf("%s-os", name)
	osVolumeDescription := fmt.Sprintf("OS Volume for %s", name)
	klog.Debugf("Creating OS volume for %s", name)
	osVolume, err := NewVolume(projectId, poolId, templateId, osVolumeName, osVolumeDescription, VolumeTypeOs, disk)
	if err != nil {
		return nil, err
	}
	volumes = append(volumes, osVolume.String())

	// create an optional data volume
	var dataVolume *Volume
	if data > 0 {
		dataVolumeName := fmt.Sprintf("%s-data", name)
		dataVolumeDescription := fmt.Sprintf("Data Volume for %s", name)
		klog.Debugf("Creating data volume for %s", name)
		dataVolume, err = NewVolume(projectId, poolId, "", dataVolumeName, dataVolumeDescription, VolumeTypeRaw, data)
		if err != nil {
			_ = osVolume.Delete()
			return nil, err
		}
		volumes = append(volumes, dataVolume.String())
	}

	adapters := []string{}

	// create a public network interface, optionnally exposed over Internet (i.e. with assigned public IPv4)
	publicAdapterName := fmt.Sprintf("%s-public-adapter", name)
	publicAdapterDescription := fmt.Sprintf("Public network adapter for %s", name)
	klog.Debugf("Creating public adapter for %s on subnet %s", name, publicSubnetId)
	publicAdapter, err := NewAdapter(publicSubnetId, publicAdapterName, publicAdapterDescription, "", []string{}, false, public)
	if err != nil {
		_ = osVolume.Delete()
		if dataVolume != nil {
			_ = dataVolume.Delete()
		}
		return nil, err
	}
	adapters = append(adapters, publicAdapter.String())

	// create a private network interface on the requested subnet (auto-assgined private IPv4)
	privateAdapterName := fmt.Sprintf("%s-private-adapter", name)
	privateAdapterDescription := fmt.Sprintf("Private network adapter for %s", name)
	klog.Debugf("Creating private adapter for %s on subnet %s", name, privateSubnetId)
	privateAdapter, err := NewAdapter(privateSubnetId, privateAdapterName, privateAdapterDescription, "", []string{}, false, true)
	if err != nil {
		_ = osVolume.Delete()
		if dataVolume != nil {
			_ = dataVolume.Delete()
		}
		_ = publicAdapter.Delete()
		return nil, err
	}
	adapters = append(adapters, privateAdapter.String())

	// optionally create extra private network interfaces on peering subnets for Kawaii (auto-assigned private IPv4)
	for _, spId := range subnetPeerings {
		spAdapterName := fmt.Sprintf("%s-peering-private-adapter", name)
		spAdapterDescription := fmt.Sprintf("Private peering network adapter for %s", name)
		klog.Debugf("Creating peering private adapter for %s on subnet %s", name, spId)
		spAdapter, err := NewAdapter(spId, spAdapterName, spAdapterDescription, "", []string{}, false, true)
		if err != nil {
			_ = osVolume.Delete()
			if dataVolume != nil {
				_ = dataVolume.Delete()
			}
			for _, adapterId := range adapters {
				a, err := FindAdapterByID(adapterId)
				if err != nil {
					_ = a.Delete()
				}
			}

			return nil, err
		}
		adapters = append(adapters, spAdapter.String())
	}

	// create an new instance with all associated settings
	klog.Debugf("Creating new instance for %s", name)
	instance, err := NewInstance(projectId, hostId, name, desc, profile, profileId, cpu, mem, adapters, volumes)
	if err != nil {
		_ = osVolume.Delete()
		if dataVolume != nil {
			_ = dataVolume.Delete()
		}
		for _, adapterId := range adapters {
			a, err := FindAdapterByID(adapterId)
			if err != nil {
				_ = a.Delete()
			}
		}
		return nil, err
	}

	kompute := Kompute{
		Resource:   NewResource(name, desc, MongoCollectionKomputeSchemaVersion),
		ProjectID:  projectId,
		InstanceID: instance.String(),
	}

	_, err = GetDB().Insert(MongoCollectionKomputeName, kompute)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new Kompute virtual machine %s", kompute.String())

	// read project object back, as it's been updated when creating volumes and instances
	prj, err = FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	// add Kompute to project
	prj.AddKompute(kompute.String())

	return &kompute, nil
}

func FindKomputes() []Kompute {
	return FindResources[Kompute](MongoCollectionKomputeName)
}

func FindKomputesByProject(projectId string) ([]Kompute, error) {
	return FindResourcesByKey[Kompute](MongoCollectionKomputeName, "project_id", projectId)
}

func FindKomputeByID(id string) (*Kompute, error) {
	return FindResourceByID[Kompute](MongoCollectionKomputeName, id)
}

func FindKomputeByName(name string) (*Kompute, error) {
	return FindResourceByName[Kompute](MongoCollectionKomputeName, name)
}

func (k *Kompute) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKomputeName, k.ID, from, to)
}

func (k *Kompute) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKomputeName, k.ID, version)
}

func (k *Kompute) migrateSchemaV2() error {
	err := k.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("instance", "instance_id")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kompute) Update(name, desc string, cpu, mem, disk, data int64) error {
	k.UpdateResourceDefaults(name, desc)

	i, err := k.Instance()
	if err != nil {
		return err
	}

	// update disk sizes, if needed
	osDiskDevice := fmt.Sprintf("%sa", VolumeOsDiskPrefix)
	osDiskId, ok := i.Disks[osDiskDevice]
	if ok {
		osDisk, err := FindVolumeByID(osDiskId)
		if err != nil {
			return err
		}
		err = osDisk.Update(osDisk.Name, osDisk.Description, disk)
		if err != nil {
			return err
		}
	}

	dataDiskDevice := fmt.Sprintf("%sb", VolumeOsDiskPrefix)
	dataDiskId, ok := i.Disks[dataDiskDevice]
	if ok {
		dataDisk, err := FindVolumeByID(dataDiskId)
		if err != nil {
			return err
		}
		err = dataDisk.Update(dataDisk.Name, dataDisk.Description, data)
		if err != nil {
			return err
		}
	}

	// update instance
	err = i.Update(name, desc, cpu, mem, i.Adapters(), i.Volumes())
	if err != nil {
		return err
	}

	k.Save()
	return nil
}

func (k *Kompute) Project() (*Project, error) {
	return FindProjectByID(k.ProjectID)
}

func (k *Kompute) Instance() (*Instance, error) {
	return FindInstanceByID(k.InstanceID)
}

func (k *Kompute) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKomputeName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Kompute) Delete() error {
	klog.Debugf("Deleting Kompute %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	// find associated instance
	i, err := k.Instance()
	if err != nil {
		klog.Error(err)
		return err
	}

	// shutdown instance
	err = i.Stop()
	if err != nil {
		klog.Error(err)
		// not a blocker
	}

	// destroy and remove associated adapters
	for _, adapterId := range i.Adapters() {
		a, err := FindAdapterByID(adapterId)
		if err != nil {
			klog.Error(err)
			continue
		}

		// remove adapter
		err = a.Delete()
		if err != nil {
			klog.Error(err)
			return err
		}
	}

	// destroy and remove associated volumes/disks
	for _, volumeId := range i.Volumes() {
		v, err := FindVolumeByID(volumeId)
		if err != nil {
			klog.Error(err)
			continue
		}

		// remove volume
		err = v.Delete()
		if err != nil {
			klog.Error(err)
			return err
		}
	}

	// remove instance
	err = i.Delete()
	if err != nil {
		klog.Error(err)
		return err
	}

	// remove kompute's reference from parents
	prj, err := k.Project()
	if err != nil {
		return err
	}
	prj.RemoveKompute(k.String())

	return GetDB().Delete(MongoCollectionKomputeName, k.ID)
}

func (k *Kompute) Model() sdk.Kompute {
	kompute := sdk.Kompute{
		Id:          k.String(),
		Name:        k.Name,
		Description: k.Description,
	}

	i, err := k.Instance()
	if err != nil {
		return kompute
	}

	kompute.Vcpus = i.CPU
	kompute.Memory = i.Memory
	kompute.Ip = i.GetIpAddress(true)

	// Kompute virtual machines only have a max of 2 disks
	osDiskDevice := fmt.Sprintf("%sa", VolumeOsDiskPrefix)
	osDiskId, ok := i.Disks[osDiskDevice]
	if ok {
		osDisk, err := FindVolumeByID(osDiskId)
		if err != nil {
			return kompute
		}
		kompute.Disk = osDisk.Size
	}

	dataDiskDevice := fmt.Sprintf("%sb", VolumeOsDiskPrefix)
	dataDiskId, ok := i.Disks[dataDiskDevice]
	if ok {
		dataDisk, err := FindVolumeByID(dataDiskId)
		if err != nil {
			return kompute
		}
		kompute.DataDisk = dataDisk.Size
	}

	return kompute
}

func (k *Kompute) GetState() (sdk.InstanceState, error) {
	i, err := k.Instance()
	if err != nil {
		return sdk.InstanceState{}, err
	}
	return i.GetState()
}

// Software OS reboot
func (k *Kompute) Reboot() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Reboot()
}

// Hardware Reset
func (k *Kompute) Reset() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Reset()
}

// Software PM Suspend
func (k *Kompute) Suspend() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Suspend()
}

// Software PM Resume
func (k *Kompute) Resume() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Resume()
}

// Enable auto-start
func (k *Kompute) AutoStart() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.AutoStart()
}

// Hardware Boot
func (k *Kompute) Start() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Start()
}

// Hardware Shutdown
func (k *Kompute) Stop() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Stop()
}

// Software Shutdown
func (k *Kompute) Shutdown() error {
	i, err := k.Instance()
	if err != nil {
		return err
	}
	return i.Shutdown()
}
