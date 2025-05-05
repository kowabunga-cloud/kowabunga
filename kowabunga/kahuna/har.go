/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

/*
 * A Kowabunga Highly-Available Resource (HAR) is a special instantiation of Kompute instances
 * existing in all of a given region's zones.
 * It provides a zone-level local service instance, allowing for local network affinity,
 * while ensuring region-level service availability through usage of cross-zones virtual IP addresses.
 */

import (
	"fmt"
	"slices"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	MongoCollectionHarSchemaVersion = 2
	MongoCollectionHarName          = "har"
)

type HighlyAvailableResource struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID       string `bson:"project_id"`
	RegionID        string `bson:"region_id"`
	PrivateSubnetID string `bson:"private_subnet_id"`

	// properties
	Profile          string    `bson:"profile"`
	PrivateAdapterID string    `bson:"private_adapter_id"`
	PrivateVIP       string    `bson:"private_vip"`
	VirtualIP        VirtualIP `bson:"virtual_ip"`
	KomputeIDs       []string  `bson:"kompute_ids"`
}

func HarMigrateSchema() error {
	for _, har := range FindHARs() {
		if har.SchemaVersion == 0 || har.SchemaVersion == 1 {
			err := har.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func FindHARs() []HighlyAvailableResource {
	return FindResources[HighlyAvailableResource](MongoCollectionHarName)
}

func FindHARByID(id string) (*HighlyAvailableResource, error) {
	return FindResourceByID[HighlyAvailableResource](MongoCollectionHarName, id)
}

func FindHARByName(name string) (*HighlyAvailableResource, error) {
	return FindResourceByName[HighlyAvailableResource](MongoCollectionHarName, name)
}

func NewHighAvailableResource(projectId, regionId, namePrefix, desc, profile, profileId string, cpu, mem, disk, data int64, kaktusIds []string) (*HighlyAvailableResource, error) {

	// find parent objects, allows to bail before creating anything
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	r, err := FindRegionByID(regionId)
	if err != nil {
		return nil, err
	}

	// use region's default pool
	poolId := r.Defaults.StoragePoolID

	// ensure pool exists
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return nil, err
	}

	// find private subnet
	privateSubnetId, err := prj.GetPrivateSubnet(regionId)
	if err != nil {
		return nil, err
	}

	// use default linux template for resources
	templateId := p.Defaults.TemplateIDs.OS

	har := HighlyAvailableResource{
		Resource:         NewResource(namePrefix, desc, MongoCollectionHarSchemaVersion),
		ProjectID:        projectId,
		RegionID:         regionId,
		PrivateSubnetID:  privateSubnetId,
		Profile:          profile,
		PrivateAdapterID: "",
		PrivateVIP:       "",
		VirtualIP:        VirtualIP{},
	}

	// HAR must reserve (and bind) private virtual IP.
	err = har.RequestPrivateVIP()
	if err != nil {
		return nil, err
	}

	har.GetVirtualIP()

	// find how many zones we're spread one (defines naming convention)
	zones := []string{}
	for _, kaktusId := range kaktusIds {
		h, err := FindKaktusByID(kaktusId)
		if err != nil {
			return nil, err
		}

		z, err := h.Zone()
		if err != nil {
			return nil, err
		}

		if !slices.Contains(zones, z.Name) {
			zones = append(zones, z.Name)
		}
	}

	// create a Kompute instance on each specified kaktus
	komputes := []string{}
	count := 1
	for _, kaktusId := range kaktusIds {
		h, err := FindKaktusByID(kaktusId)
		if err != nil {
			return nil, err
		}

		z, err := h.Zone()
		if err != nil {
			return nil, err
		}

		harName := fmt.Sprintf("%s-%s", namePrefix, z.Name)
		name := fmt.Sprintf("%s-%d", harName, count)
		if len(zones) == 1 {
			// resources spread against a single zone, increase suffix counter
			count += 1
		}

		// spin-up instance
		kompute, err := NewKompute(projectId, z.String(), h.String(), poolId, templateId,
			name, desc, har.Profile, profileId, cpu, mem, disk, 0, false, []string{}, nil)
		if err != nil {
			for _, komputeId := range komputes {
				komputeToDel, err := FindKomputeByID(komputeId)
				if err != nil {
					return nil, err
				}
				err = komputeToDel.Delete()
				if err != nil {
					klog.Error("Cleaning underlying Kompute " + komputeId + "failed. Continue cleaning " + err.Error() + ". ")
				}
			}
			return nil, err
		}

		komputes = append(komputes, kompute.String())
	}
	har.KomputeIDs = komputes

	klog.Debugf("Created new Highly-Available Resource %s", har.String())
	_, err = GetDB().Insert(MongoCollectionHarName, har)
	if err != nil {
		return nil, err
	}

	return &har, nil
}

func (har *HighlyAvailableResource) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionHarName, har.ID, from, to)
}

func (har *HighlyAvailableResource) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionHarName, har.ID, version)
}

func (har *HighlyAvailableResource) migrateSchemaV2() error {
	err := har.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = har.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = har.renameDbField("private_subnet", "private_subnet_id")
	if err != nil {
		return err
	}

	err = har.renameDbField("private_adapter", "private_adapter_id")
	if err != nil {
		return err
	}

	err = har.renameDbField("kces", "kompute_ids")
	if err != nil {
		return err
	}

	err = har.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (har *HighlyAvailableResource) Project() (*Project, error) {
	return FindProjectByID(har.ProjectID)
}

func (har *HighlyAvailableResource) RequestPrivateVIP() error {
	r, err := FindRegionByID(har.RegionID)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%s-%s", har.Name, r.Name)
	adapterName := fmt.Sprintf("%s-VIP-private-adapter-0", prefix)
	adapterDesc := fmt.Sprintf("private network adapter for %s", prefix)

	klog.Debugf("Creating private VIP adapter for %s on subnet %s", prefix, har.PrivateSubnetID)
	adapter, err := NewAdapter(har.PrivateSubnetID, adapterName, adapterDesc, "", []string{}, false, true)
	if err != nil {
		return err
	}

	har.PrivateAdapterID = adapter.String()
	har.PrivateVIP = adapter.Addresses[0]

	return nil
}

func (har *HighlyAvailableResource) GetVirtualIP() {
	// perform various checks before going any further ...
	privateInterface := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex+1)
	privateSubnet, err := FindSubnetByID(har.PrivateSubnetID)
	if err != nil {
		return
	}

	prj, err := har.Project()
	if err != nil {
		return
	}

	vrid, err := prj.AllocateVRID()
	if err != nil {
		return
	}

	vip := VirtualIP{
		VRRP:        vrid,
		Interface:   privateInterface,
		VIP:         har.PrivateVIP,
		Priority:    VrrpPriorityBackup,
		NetMaskSize: privateSubnet.Size(),
		Public:      false,
	}
	har.VirtualIP = vip
}

func (har *HighlyAvailableResource) Delete() error {

	klog.Debugf("Deleting HAR Kompute instances %s", har.String())

	// Destroy underlying Komputes
	for _, komputeId := range har.KomputeIDs {
		kompute, err := FindKomputeByID(komputeId)
		if err != nil {
			klog.Error(err)
			return err
		}
		err = kompute.Delete()
		if err != nil {
			return err
		}
	}

	// Destroy VIP adapter
	a, err := FindAdapterByID(har.PrivateAdapterID)
	if err != nil {
		return err
	}
	err = a.Delete()
	if err != nil {
		return err
	}

	// drop VRRP ID from project
	prj, err := har.Project()
	if err != nil {
		return err
	}
	prj.RemoveVRID(har.VirtualIP.VRRP)

	return GetDB().Delete(MongoCollectionHarName, har.ID)
}
