/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kiwi"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionKiwiSchemaVersion = 2
	MongoCollectionKiwiName          = "kiwi"
)

type Kiwi struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	RegionID string `bson:"region_id"`

	// properties
	AgentIDs []string `bson:"agent_ids"`

	// children references
}

func KiwiMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("netgws", MongoCollectionKiwiName)
	if err != nil {
		return err
	}

	for _, kiwi := range FindKiwis() {
		if kiwi.SchemaVersion == 0 || kiwi.SchemaVersion == 1 {
			err := kiwi.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewKiwi(regionId, name, desc string, agts []string) (*Kiwi, error) {
	gw := Kiwi{
		Resource: NewResource(name, desc, MongoCollectionKiwiSchemaVersion),
		RegionID: regionId,
		AgentIDs: VerifyAgents(agts, common.KowabungaKiwiAgent),
	}

	r, err := gw.Region()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionKiwiName, gw)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new network gateway %s", gw.String())

	// add network gateway to region
	r.AddKiwi(gw.String())

	return &gw, nil
}

func FindKiwis() []Kiwi {
	return FindResources[Kiwi](MongoCollectionKiwiName)
}

func FindKiwisByRegion(regionId string) ([]Kiwi, error) {
	return FindResourcesByKey[Kiwi](MongoCollectionKiwiName, "region_id", regionId)
}

func FindKiwiByID(id string) (*Kiwi, error) {
	return FindResourceByID[Kiwi](MongoCollectionKiwiName, id)
}

func FindKiwiByName(name string) (*Kiwi, error) {
	return FindResourceByName[Kiwi](MongoCollectionKiwiName, name)
}

func (k *Kiwi) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKiwiName, k.ID, from, to)
}

func (k *Kiwi) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKiwiName, k.ID, version)
}

func (k *Kiwi) migrateSchemaV2() error {
	err := k.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("agents", "agent_ids")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kiwi) Region() (*Region, error) {
	return FindRegionByID(k.RegionID)
}

func (k *Kiwi) RPC(method string, args, reply any) error {
	return RPC(k.AgentIDs, method, args, reply)
}

func (k *Kiwi) Update(name, desc string, agts []string) {
	k.UpdateResourceDefaults(name, desc)

	k.AgentIDs = VerifyAgents(agts, common.KowabungaKiwiAgent)

	k.Save()
}

func (k *Kiwi) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKiwiName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Kiwi) Delete() error {
	klog.Debugf("Deleting network gateway %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	// remove region's reference from parents
	r, err := k.Region()
	if err != nil {
		return err
	}
	r.RemoveKiwi(k.String())

	return GetDB().Delete(MongoCollectionKiwiName, k.ID)
}

func (k *Kiwi) Model() sdk.Kiwi {
	return sdk.Kiwi{
		Id:          k.String(),
		Name:        k.Name,
		Description: k.Description,
		Agents:      k.AgentIDs,
	}
}

func (k *Kiwi) Reload() error {
	args := kiwi.KiwiReloadArgs{}

	projects := FindProjects()
	for _, p := range projects {

		domain := kiwi.KiwiReloadArgsDomain{
			Name: p.Domain,
		}

		recordIds := p.DnsRecords()
		for _, pid := range recordIds {
			record, err := p.FindDnsRecordByID(pid)
			if err != nil {
				return err
			}

			r := kiwi.KiwiReloadArgsRecord{
				Name:      record.Name,
				Type:      "A",
				Addresses: record.Addresses,
			}
			domain.Records = append(domain.Records, r)
		}

		args.Domains = append(args.Domains, domain)
	}

	region, err := k.Region()
	if err != nil {
		return err
	}

	recordIds := region.DnsRecords()
	for _, pid := range recordIds {
		record, err := region.FindDnsRecordByID(pid)
		if err != nil {
			return err
		}

		domain := kiwi.KiwiReloadArgsDomain{
			Name: region.Domain,
		}

		r := kiwi.KiwiReloadArgsRecord{
			Name:      record.Name,
			Type:      "A",
			Addresses: record.Addresses,
		}
		domain.Records = append(domain.Records, r)
		args.Domains = append(args.Domains, domain)
	}

	var reply kiwi.KiwiReloadReply

	return k.RPC("Reload", args, &reply)
}

func (k *Kiwi) CreateDnsZone(domain string) error {
	args := kiwi.KiwiCreateDnsZoneArgs{
		Domain: domain,
	}
	var reply kiwi.KiwiCreateDnsZoneReply

	return k.RPC("CreateDnsZone", args, &reply)
}

func (k *Kiwi) DeleteDnsZone(domain string) error {
	args := kiwi.KiwiDeleteDnsZoneArgs{
		Domain: domain,
	}
	var reply kiwi.KiwiDeleteDnsZoneReply

	return k.RPC("DeleteDnsZone", args, &reply)
}

func (k *Kiwi) CreateDnsRecord(domain, entry string, addr []string) error {
	args := kiwi.KiwiCreateDnsRecordArgs{
		Domain:    domain,
		Entry:     entry,
		Addresses: addr,
	}
	var reply kiwi.KiwiCreateDnsRecordReply

	return k.RPC("CreateDnsRecord", args, &reply)
}

func (k *Kiwi) UpdateDnsRecord(domain, entry string, addr []string) error {
	args := kiwi.KiwiUpdateDnsRecordArgs{
		Domain:    domain,
		Entry:     entry,
		Addresses: addr,
	}
	var reply kiwi.KiwiUpdateDnsRecordReply

	return k.RPC("UpdateDnsRecord", args, &reply)
}

func (k *Kiwi) DeleteDnsRecord(domain, entry string) error {
	args := kiwi.KiwiDeleteDnsRecordArgs{
		Domain: domain,
		Entry:  entry,
	}
	var reply kiwi.KiwiDeleteDnsRecordReply

	return k.RPC("DeleteDnsRecord", args, &reply)
}
