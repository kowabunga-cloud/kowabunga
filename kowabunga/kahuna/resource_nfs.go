/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"crypto/rand"
	"math/big"
	"slices"
	"strconv"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionNfsSchemaVersion = 2
	MongoCollectionNfsName          = "nfs"

	NfsBackendApiPort        = 54934
	NfsFileSystemNameDefault = "nfs"
	ErrNfsNoSuchKylo         = "no such Kylo in NFS storage"
	NfsExportIdMin           = 100
	NfsExportIdMax           = 65000
)

type NFS struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	RegionID      string `bson:"region_id"`
	StoragePoolID string `bson:"storage_pool_id"`

	// properties
	Endpoint string        `bson:"endpoint"`
	FS       string        `bson:"fs"`
	Ganesha  NfsGaneshaAPI `bson:"ganesha"`

	// children references
	KyloIDs []string `bson:"kylo_ids"`
	Exports []string `bson:"exports"`
}

type NfsGaneshaAPI struct {
	Backends []string `bson:"backends"`
	Port     int      `bson:"port"`
}

func NfsMigrateSchema() error {
	for _, nfs := range FindNFSes() {
		if nfs.SchemaVersion == 0 || nfs.SchemaVersion == 1 {
			err := nfs.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewNfs(regionId, poolId, name, desc, endpoint, fs string, backends []string, port int) (*NFS, error) {
	n := NFS{
		Resource:      NewResource(name, desc, MongoCollectionNfsSchemaVersion),
		RegionID:      regionId,
		StoragePoolID: poolId,
		Endpoint:      endpoint,

		FS: fs,
		Ganesha: NfsGaneshaAPI{
			Backends: backends,
			Port:     port,
		},
		KyloIDs: []string{},
	}

	if port == 0 {
		n.Ganesha.Port = NfsBackendApiPort
	}

	r, err := n.Region()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionNfsName, n)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new NFS storage %s", n.String())

	// add NFS storage to region
	r.AddNfs(n.String())

	return &n, nil
}

func FindNFSes() []NFS {
	return FindResources[NFS](MongoCollectionNfsName)
}

func FindNfsByRegion(regionId string) ([]NFS, error) {
	return FindResourcesByKey[NFS](MongoCollectionNfsName, "region_id", regionId)
}

func FindNfsByID(id string) (*NFS, error) {
	return FindResourceByID[NFS](MongoCollectionNfsName, id)
}

func FindNfsByName(name string) (*NFS, error) {
	return FindResourceByName[NFS](MongoCollectionNfsName, name)
}

func (n *NFS) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionNfsName, n.ID, from, to)
}

func (n *NFS) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionNfsName, n.ID, version)
}

func (n *NFS) migrateSchemaV2() error {
	err := n.renameDbField("region", "region_id")
	if err != nil {
		return err
	}

	err = n.renameDbField("pool", "storage_pool_id")
	if err != nil {
		return err
	}

	err = n.renameDbField("kfs", "kylo_ids")
	if err != nil {
		return err
	}

	err = n.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (n *NFS) Region() (*Region, error) {
	return FindRegionByID(n.RegionID)
}

func (n *NFS) StoragePool() (*StoragePool, error) {
	return FindStoragePoolByID(n.StoragePoolID)
}

func (n *NFS) RPC(method string, args, reply any) error {
	p, err := n.StoragePool()
	if err != nil {
		return err
	}

	return p.RPC(method, args, reply)
}

func (n *NFS) HasChildren() bool {
	return HasChildRefs(n.KyloIDs)
}

func (n *NFS) FindKylo() ([]Kylo, error) {
	return FindKyloByNfs(n.String())
}

func (n *NFS) Update(name, desc, endpoint, fs string, backends []string, port int) {
	n.UpdateResourceDefaults(name, desc)

	SetFieldStr(&n.Endpoint, endpoint)
	SetFieldStr(&n.FS, fs)
	n.Ganesha.Backends = backends
	n.Ganesha.Port = port
	if port == 0 {
		n.Ganesha.Port = NfsBackendApiPort
	}

	n.Save()
}

func (n *NFS) Save() {
	n.Updated()
	_, err := GetDB().Update(MongoCollectionNfsName, n.ID, n)
	if err != nil {
		klog.Error(err)
	}
}

func (n *NFS) Delete() error {
	klog.Debugf("Deleting NFS storage %s", n.String())

	if n.String() == ResourceUnknown {
		return nil
	}

	// remove region's reference from parents
	r, err := n.Region()
	if err != nil {
		return err
	}
	r.RemoveNfs(n.String())

	return GetDB().Delete(MongoCollectionNfsName, n.ID)
}

func (n *NFS) Model() sdk.StorageNfs {
	port := int64(n.Ganesha.Port)
	return sdk.StorageNfs{
		Id:          n.String(),
		Name:        n.Name,
		Description: n.Description,
		Endpoint:    n.Endpoint,
		Fs:          n.FS,
		Backends:    n.Ganesha.Backends,
		Port:        port,
	}
}

// Kylo
func (n *NFS) Kylos() []string {
	return n.KyloIDs
}

func (n *NFS) Kylo(id string) (*Kylo, error) {
	return FindChildByID[Kylo](&n.KyloIDs, id, MongoCollectionKyloName, ErrNfsNoSuchKylo)
}

func (n *NFS) AddKylo(id, exportId string) {
	klog.Debugf("Adding Kylo %s to NFS storage %s", id, n.String())
	AddChildRef(&n.KyloIDs, id)
	AddChildRef(&n.Exports, exportId)
	n.Save()
}

func (n *NFS) RemoveKylo(id, exportId string) {
	klog.Debugf("Removing Kylo %s from NFS storage %s", id, n.String())
	RemoveChildRef(&n.KyloIDs, id)
	RemoveChildRef(&n.Exports, exportId)
	n.Save()
}

func (n *NFS) NewExportId() string {
	for {
		rId, err := rand.Int(rand.Reader, big.NewInt(NfsExportIdMax-NfsExportIdMin+1))
		if err != nil {
			continue
		}

		id := rId.Int64() + NfsExportIdMin
		idStr := strconv.FormatInt(id, 10)

		if slices.Contains(n.Exports, idStr) {
			continue
		}

		return idStr
	}
}
