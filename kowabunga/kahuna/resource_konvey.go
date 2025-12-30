/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/agents"
	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/metadata"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionKonveySchemaVersion = 2
	MongoCollectionKonveyName          = "konvey"

	KonveyCpu    = 1
	KonveyMemory = 1 * common.GiB  // 4GB
	KonveyDisk   = 16 * common.GiB // 16GB
	// Konvey name will always result in konvey-<regionname>
	KonveyDefaultNamePrefix = "konvey"
)

type Konvey struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`

	// properties
	Failover  bool             `bson:"failover"`
	Endpoints []KonveyEndpoint `bson:"endpoints"`

	// children references
	HighlyAvailableResourceID string `bson:"har_id"`
}

type KonveyEndpoint struct {
	Name     string          `bson:"name"`
	Port     int64           `bson:"port"`
	Protocol string          `bson:"protocol"`
	Backends []KonveyBackend `bson:"backends"`
}

type KonveyBackend struct {
	Host string `bson:"host"`
	Port int64  `bson:"port"`
}

func (ke *KonveyEndpoint) Model() sdk.KonveyEndpoint {
	ep := sdk.KonveyEndpoint{
		Name:     ke.Name,
		Port:     ke.Port,
		Protocol: ke.Protocol,
		Backends: sdk.KonveyBackends{},
	}

	for _, backend := range ke.Backends {
		ep.Backends.Hosts = append(ep.Backends.Hosts, backend.Host)
		ep.Backends.Port = backend.Port
	}

	return ep
}

func (ke *KonveyEndpoint) Metadata() metadata.KonveyEndpointMetadata {
	m := metadata.KonveyEndpointMetadata{
		Name:     ke.Name,
		Port:     ke.Port,
		Protocol: ke.Protocol,
		Backends: []metadata.KonveyBackendMetadata{},
	}

	for _, backend := range ke.Backends {
		b := metadata.KonveyBackendMetadata{
			Host: backend.Host,
			Port: backend.Port,
		}
		m.Backends = append(m.Backends, b)
	}

	return m
}

func KonveyMigrateSchema() error {
	for _, konvey := range FindKonveys() {
		if konvey.SchemaVersion == 0 || konvey.SchemaVersion == 1 {
			err := konvey.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewKonvey(projectId, regionId, name, desc string, endpoints []KonveyEndpoint, hostIds []string) (*Konvey, error) {

	klog.Debug("Creating underlying HAR resources for Konvey")

	k := Konvey{
		Resource:  NewResource(name, desc, MongoCollectionKonveySchemaVersion),
		ProjectID: projectId,
		Failover:  false,
		Endpoints: endpoints,
	}

	if len(hostIds) > 1 {
		k.Failover = true
	}

	har, err := NewHighAvailableResource(projectId, regionId, name, desc, CloudinitProfileKonvey, k.String(), KonveyCpu, KonveyMemory, KonveyDisk, 0, hostIds)
	if err != nil {
		return nil, err
	}
	k.HighlyAvailableResourceID = har.String()

	klog.Debugf("Created new Konvey %s", k.String())
	_, err = GetDB().Insert(MongoCollectionKonveyName, k)
	if err != nil {
		return nil, err
	}

	// read project object back, as it's been updated
	prj, err := FindProjectByID(projectId)
	if err != nil {
		return nil, err
	}

	// add Konvey to project
	prj.AddKonvey(k.String())

	return &k, err
}

func FindKonveys() []Konvey {
	return FindResources[Konvey](MongoCollectionKonveyName)
}

func FindKonveysByProject(projectId string) ([]Konvey, error) {
	return FindResourcesByKey[Konvey](MongoCollectionKonveyName, "project_id", projectId)
}

func FindKonveyByID(id string) (*Konvey, error) {
	return FindResourceByID[Konvey](MongoCollectionKonveyName, id)
}

func FindKonveyByName(name string) (*Konvey, error) {
	return FindResourceByName[Konvey](MongoCollectionKonveyName, name)
}

func (k *Konvey) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKonveyName, k.ID, from, to)
}

func (k *Konvey) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKonveyName, k.ID, version)
}

func (k *Konvey) migrateSchemaV2() error {
	err := k.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("har", "har_id")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Konvey) Project() (*Project, error) {
	prj, err := FindProjectByID(k.ProjectID)
	if err != nil {
		return nil, err
	}
	return prj, nil
}

func (k *Konvey) HAR() (*HighlyAvailableResource, error) {
	return FindHARByID(k.HighlyAvailableResourceID)
}

func (k *Konvey) Update(desc string, endpoints []KonveyEndpoint) error {
	k.Description = desc
	k.Endpoints = endpoints
	k.Save()

	har, err := k.HAR()
	if err != nil {
		return nil // bypass error
	}

	for _, komputeId := range har.KomputeIDs {
		kompute, err := FindKomputeByID(komputeId)
		if err != nil {
			continue
		}

		i, err := kompute.Instance()
		if err != nil {
			continue
		}

		args := agents.KontrollerReloadArgs{}
		var reply agents.KontrollerReloadReply
		err = i.InstanceRPC("Reload", args, &reply)
		if err != nil {
			continue
		}
	}

	return nil
}

func (k *Konvey) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKonveyName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Konvey) Delete() error {
	klog.Debugf("Deleting Konvey %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	har, err := k.HAR()
	if err != nil {
		klog.Error(err)
		return err
	}
	err = har.Delete()
	if err != nil {
		return err
	}

	// remove konvey's reference from parents
	prj, err := k.Project()
	if err != nil {
		return err
	}
	prj.RemoveKonvey(k.String())

	return GetDB().Delete(MongoCollectionKonveyName, k.ID)
}

func (k *Konvey) Model() sdk.Konvey {
	konvey := sdk.Konvey{
		Id:          k.String(),
		Name:        strings.TrimPrefix(k.Name, fmt.Sprintf("%s-", KonveyDefaultNamePrefix)),
		Description: k.Description,
		Failover:    k.Failover,
	}

	har, err := k.HAR()
	if err != nil {
		return konvey
	}

	konvey.Vip = har.VirtualIP.VIP
	for _, ep := range k.Endpoints {
		konvey.Endpoints = append(konvey.Endpoints, ep.Model())
	}

	return konvey
}

func (k *Konvey) Metadata(instanceId string) metadata.KonveyMetadata {
	privateInterface := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex+1)

	meta := metadata.KonveyMetadata{
		PrivateInterface:     privateInterface,
		VrrpControlInterface: privateInterface,
		VirtualIPs:           []metadata.VirtualIpMetadata{},
		Endpoints:            []metadata.KonveyEndpointMetadata{},
	}

	for _, ep := range k.Endpoints {
		meta.Endpoints = append(meta.Endpoints, ep.Metadata())
	}

	har, err := k.HAR()
	if err != nil {
		return meta
	}

	meta.VirtualIPs = append(meta.VirtualIPs, har.VirtualIP.Metadata())

	// tune-in VRRP priority
	sort.Strings(har.KomputeIDs)
	if len(har.KomputeIDs) > 0 {
		for id := range meta.VirtualIPs {
			meta.VirtualIPs[id].Priority = VrrpPriorityBackup
			if har.KomputeIDs[0] == instanceId {
				meta.VirtualIPs[id].Priority = VrrpPriorityMaster
			}
		}
	}

	return meta
}
