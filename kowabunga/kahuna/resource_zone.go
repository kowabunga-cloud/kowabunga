/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/agnivade/levenshtein"
	"github.com/huandu/xstrings"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionZoneSchemaVersion = 2
	MongoCollectionZoneName          = "zone"

	ErrZoneNoSuchKaktus = "no such kaktus in zone"
)

type Zone struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	RegionID string `bson:"region_id"`

	// properties
	VirtualResources ZoneVirtualResources `bson:"virtual_resources"`

	// children references
	KaktusIDs []string `bson:"kaktus_ids"`
}

type ZoneVirtualResources struct {
	Computing ZoneVirtualResource `bson:"vcpu"`
	Memory    ZoneVirtualResource `bson:"memory_gb"`
}

type ZoneVirtualResource struct {
	Count    int64   `bson:"count"`
	Price    float32 `bson:"price,truncate"`
	Currency string  `bson:"currency"`
}

func ZoneMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("zones", MongoCollectionZoneName)
	if err != nil {
		return err
	}

	for _, zone := range FindZones() {
		if zone.SchemaVersion == 0 || zone.SchemaVersion == 1 {
			err := zone.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewZone(regionId, name, desc string) (*Zone, error) {

	// create resource
	z := Zone{
		Resource:         NewResource(name, desc, MongoCollectionZoneSchemaVersion),
		RegionID:         regionId,
		VirtualResources: ZoneVirtualResources{},
		KaktusIDs:        []string{},
	}

	r, err := z.Region()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionZoneName, z)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new zone %s", z.String())

	// add zone to region
	r.AddZone(z.String())

	return &z, nil
}

func FindZones() []Zone {
	return FindResources[Zone](MongoCollectionZoneName)
}

func FindZonesByRegion(regionId string) ([]Zone, error) {
	return FindResourcesByKey[Zone](MongoCollectionZoneName, "region_id", regionId)
}

func FindZoneByID(id string) (*Zone, error) {
	return FindResourceByID[Zone](MongoCollectionZoneName, id)
}

func FindZoneByName(name string) (*Zone, error) {
	return FindResourceByName[Zone](MongoCollectionZoneName, name)
}

func (z *Zone) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionZoneName, z.ID, from, to)
}

func (z *Zone) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionZoneName, z.ID, version)
}

func (z *Zone) migrateSchemaV2() error {
	err := z.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = z.renameDbField("hosts", "kaktus_ids")
	if err != nil {
		return err
	}

	err = z.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (z *Zone) Region() (*Region, error) {
	return FindRegionByID(z.RegionID)
}

func (z *Zone) AverageVirtualResources() *ZoneVirtualResources {
	return &z.VirtualResources
}

func (z *Zone) HasChildren() bool {
	return HasChildRefs(z.KaktusIDs)
}

func (z *Zone) FindKaktus() ([]Kaktus, error) {
	return FindKaktusByZone(z.String())
}

func (z *Zone) Update(name, desc string) {
	z.UpdateResourceDefaults(name, desc)
	z.Save()
}

func (z *Zone) Save() {
	z.Updated()
	_, err := GetDB().Update(MongoCollectionZoneName, z.ID, z)
	if err != nil {
		klog.Error(err)
	}
}

func (z *Zone) Delete() error {
	klog.Debugf("Deleting zone %s", z.String())

	if z.String() == ResourceUnknown {
		return nil
	}

	// remove zone's reference from parents
	r, err := z.Region()
	if err != nil {
		return err
	}
	r.RemoveZone(z.String())

	return GetDB().Delete(MongoCollectionZoneName, z.ID)
}

func (z *Zone) Model() sdk.Zone {
	return sdk.Zone{
		Id:          z.String(),
		Name:        z.Name,
		Description: z.Description,
	}
}

func (z *Zone) ElectMostFavorableKaktuses(instanceName string, number int) ([]*Kaktus, error) {
	var kaktuses []*Kaktus
	kaktusCandidates := z.KaktusIDs

	for i := 0; i < number; i++ {
		kaktus, err := z.ElectMostFavorableKaktus(instanceName, kaktusCandidates)
		if err != nil {
			return kaktuses, err
		}

		var tempKaktusList []string
		for _, h := range kaktusCandidates {
			if kaktus.String() != h {
				tempKaktusList = append(tempKaktusList, h)
			}
		}
		kaktusCandidates = tempKaktusList
		if len(kaktusCandidates) == 0 {
			kaktusCandidates = z.KaktusIDs
		}

		kaktuses = append(kaktuses, kaktus)
	}
	return kaktuses, nil
}

func (z *Zone) ElectMostFavorableKaktus(instanceName string, kaktusCandidates []string) (*Kaktus, error) {
	var kaktus *Kaktus
	bestScore := KaktusMaxScore
	for _, id := range kaktusCandidates {
		h, err := FindKaktusByID(id)
		if err != nil {
			continue
		}

		// verify kaktus current virtual resources usage and give a score
		score := h.UsageScore()

		// now look for possibly existing instances on this kaktus with a close name (i.e. cluster siblings)
		for _, id := range h.Instances() {
			i, err := FindInstanceByID(id)
			if err != nil {
				continue
			}

			distance := levenshtein.ComputeDistance(i.Name, instanceName)
			if i.Name == xstrings.Successor(instanceName) || instanceName == xstrings.Successor(i.Name) || distance < KaktusScoreMaxLevenshteinDist {
				// apply malus
				score += KaktusScoreSiblingMalus
			}
		}

		klog.Debugf("Kaktus %s has a score of %d (best: %d)", h.String(), score, bestScore)
		if score < bestScore {
			// we got a potential winner
			bestScore = score
			kaktus = h
		}
	}

	if kaktus == nil {
		return nil, fmt.Errorf("%s", ErrZoneNoSuchKaktus)
	}

	klog.Infof("Best instance hosting candidate looks like %s (%s), with a score of %d", kaktus.String(), kaktus.Name, bestScore)

	return kaktus, nil
}

func (z *Zone) UsageScore() int {
	score := 0
	for _, id := range z.KaktusIDs {
		h, err := FindKaktusByID(id)
		if err != nil {
			continue
		}

		// adds kaktus current virtual resources usage and give a score
		score += h.UsageScore()
	}

	return score
}

// Kaktus

func (z *Zone) Kaktuses() []string {
	return z.KaktusIDs
}

func (z *Zone) Kaktus(id string) (*Kaktus, error) {
	return FindChildByID[Kaktus](&z.KaktusIDs, id, MongoCollectionKaktusName, ErrZoneNoSuchKaktus)
}

func (z *Zone) AddKaktus(id string) {
	klog.Debugf("Adding kaktus %s to zone %s", id, z.String())
	AddChildRef(&z.KaktusIDs, id)
	z.Save()
}

func (z *Zone) RemoveKaktus(id string) {
	klog.Debugf("Removing kaktus %s from zone %s", id, z.String())
	RemoveChildRef(&z.KaktusIDs, id)
	err := z.UpdateCapabilities()
	if err != nil {
		klog.Error(err.Error())
	}
	z.Save()
}

// Cost
func (z *Zone) UpdateCapabilities() error {
	klog.Debugf("Updating zone %s virtual resources capabilities", z)

	res := ZoneVirtualResources{}

	for _, kaktusId := range z.KaktusIDs {
		h, err := FindKaktusByID(kaktusId)
		if err != nil {
			return err
		}

		res.Computing.Count += h.VirtualResources.VCPU.Count
		res.Computing.Price += h.VirtualResources.VCPU.Price
		res.Computing.Currency = h.VirtualResources.VCPU.Currency
		res.Memory.Count += h.VirtualResources.VMemGB.Count
		res.Memory.Price += h.VirtualResources.VMemGB.Price
		res.Memory.Currency = h.VirtualResources.VMemGB.Currency

	}
	res.Computing.Price /= float32(len(z.KaktusIDs))
	res.Memory.Price /= float32(len(z.KaktusIDs))

	klog.Debugf("Zone %s vCPU count: %d", z, res.Computing.Count)
	klog.Debugf("Zone %s vCPU average price: %f %s", z, res.Computing.Price, res.Computing.Currency)
	klog.Debugf("Zone %s vMemory GB count: %d", z, res.Memory.Count)
	klog.Debugf("Zone %s vMemory GB average price: %f %s", z, res.Memory.Price, res.Memory.Currency)

	z.VirtualResources = res
	z.Save()

	// triggers cost recomputation of all of zone's instances
	for _, kaktusId := range z.KaktusIDs {
		h, err := FindKaktusByID(kaktusId)
		if err != nil {
			return err
		}

		for _, instanceId := range h.Instances() {
			i, err := FindInstanceByID(instanceId)
			if err != nil {
				return err
			}
			err = i.ComputeCost(&res)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
