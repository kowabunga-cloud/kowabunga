/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"net"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionDnsRecordSchemaVersion = 2
	MongoCollectionDnsRecordName          = "dns_record"
)

type DnsRecord struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`

	// properties
	Domain    string   `bson:"domain"`
	Addresses []string `bson:"addresses"`

	// children references
}

func DnsRecordMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("records", MongoCollectionDnsRecordName)
	if err != nil {
		return err
	}

	for _, record := range FindDnsRecords() {
		if record.SchemaVersion == 0 || record.SchemaVersion == 1 {
			err = record.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewDnsRecord(projectId, domain, name, desc string, addresses []string) (*DnsRecord, error) {

	// ensure we have a rightful domain, if any
	if domain != "" && !VerifyDomain(domain) {
		err := fmt.Errorf("Invalid domain name: %s", domain)
		return nil, err
	}

	// ensure we have rightful IPv4 addresses
	for _, i := range addresses {
		ip := net.ParseIP(i)
		if ip == nil {
			return nil, fmt.Errorf("Invalid IPv4 address: %s", i)
		}
	}

	r := DnsRecord{
		Resource:  NewResource(name, desc, MongoCollectionDnsRecordSchemaVersion),
		ProjectID: projectId,
		Domain:    domain,
		Addresses: addresses,
	}

	prj, err := r.Project()
	if err != nil {
		return nil, err
	}

	err = r.CreateDnsRecord()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionDnsRecordName, r)
	if err != nil {
		return nil, err
	}

	klog.Infof("Created new DNS record %s (%s.%s)", r.String(), r.Name, r.Domain)

	// add volume to project
	prj.AddDnsRecord(r.String())

	return &r, nil
}

func FindDnsRecords() []DnsRecord {
	return FindResources[DnsRecord](MongoCollectionDnsRecordName)
}

func FindRecordsByProject(projectId string) ([]DnsRecord, error) {
	return FindResourcesByKey[DnsRecord](MongoCollectionDnsRecordName, "project_id", projectId)
}

func FindDnsRecordByID(id string) (*DnsRecord, error) {
	return FindResourceByID[DnsRecord](MongoCollectionDnsRecordName, id)
}

func FindDnsRecordByDomainAndName(domain, name string) (*DnsRecord, error) {
	records := FindResources[DnsRecord](MongoCollectionDnsRecordName)
	for _, r := range records {
		if r.Name == name && r.Domain == domain {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("no such DNS record")
}

func (r *DnsRecord) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionDnsRecordName, r.ID, from, to)
}

func (r *DnsRecord) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionDnsRecordName, r.ID, version)
}

func (r *DnsRecord) migrateSchemaV2() error {
	err := r.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = r.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (r *DnsRecord) Project() (*Project, error) {
	return FindProjectByID(r.ProjectID)
}

func (r *DnsRecord) CreateDnsRecord() error {
	// create record on all possible network gateways
	for _, gw := range FindKiwis() {
		err := gw.CreateDnsRecord(r.Domain, r.Name, r.Addresses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DnsRecord) DeleteDnsRecord() error {
	// delete zone from all possible network gateways
	for _, gw := range FindKiwis() {
		err := gw.DeleteDnsRecord(r.Domain, r.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DnsRecord) Update(name, desc string, addresses []string) {
	r.UpdateResourceDefaults(name, desc)
	r.Addresses = addresses

	// update record on all possible network gateways
	for _, gw := range FindKiwis() {
		err := gw.UpdateDnsRecord(r.Domain, r.Name, r.Addresses)
		if err != nil {
			klog.Error(err)
		}
	}

	r.Save()
}

func (r *DnsRecord) Save() {
	r.Updated()
	_, err := GetDB().Update(MongoCollectionDnsRecordName, r.ID, r)
	if err != nil {
		klog.Error(err)
	}
}

func (r *DnsRecord) Delete() error {
	klog.Infof("Deleting DNS record %s (%s.%s)", r.String(), r.Name, r.Domain)

	if r.String() == ResourceUnknown {
		return nil
	}

	err := r.DeleteDnsRecord()
	if err != nil {
		return err
	}

	// remove record's reference from parents
	prj, err := r.Project()
	if err != nil {
		return err
	}
	prj.RemoveDnsRecord(r.String())

	return GetDB().Delete(MongoCollectionDnsRecordName, r.ID)
}

func (r *DnsRecord) Model() sdk.DnsRecord {
	return sdk.DnsRecord{
		Id:          r.String(),
		Name:        r.Name,
		Description: r.Description,
		Domain:      r.Domain,
		Addresses:   r.Addresses,
	}
}
