/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/proto"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionKyloSchemaVersion = 2
	MongoCollectionKyloName          = "kylo"
)

type Kylo struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`
	NfsID     string `bson:"nfs_id"`

	// properties
	SubVolume  KyloSubVolume `bson:"subvolume"`
	ExportID   string        `bson:"export_id"`
	AccessType string        `bson:"access_type"`
	Protocols  []int32       `bson:"protocols"`
	Clients    []string      `bson:"clients"`

	// children references
}

type KyloSubVolume struct {
	Name string `bson:"name"`
	Path string `bson:"path"`
	Size int64  `bson:"size"`
}

func KyloMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("kfs", MongoCollectionKyloName)
	if err != nil {
		return err
	}

	for _, kylo := range FindKylos() {
		if kylo.SchemaVersion == 0 || kylo.SchemaVersion == 1 {
			err := kylo.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (k *Kylo) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionKyloName, k.ID, from, to)
}

func (k *Kylo) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionKyloName, k.ID, version)
}

func (k *Kylo) migrateSchemaV2() error {
	err := k.renameDbField("project", "project_id")
	if err != nil {
		return err
	}

	err = k.renameDbField("nfs", "nfs_id")
	if err != nil {
		return err
	}

	err = k.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (k *Kylo) listFs() ([]string, error) {
	args := proto.KaktusListFileSystemsArgs{}
	var reply proto.KaktusListFileSystemsReply
	err := k.RPC(proto.RpcKaktusListFileSystems, args, &reply)
	if err != nil {
		return []string{}, err
	}
	return reply.FS, nil
}

func (k *Kylo) listSubVolumes(fs string) ([]string, error) {
	args := proto.KaktusListFsSubVolumesArgs{
		FS: fs,
	}
	var reply proto.KaktusListFsSubVolumesReply
	err := k.RPC(proto.RpcKaktusListFsSubVolumes, args, &reply)
	if err != nil {
		return []string{}, err
	}
	return reply.SubVolumes, nil
}

func (k *Kylo) createSubVolume(fs, vol string) (string, int64, error) {
	args := proto.KaktusCreateFsSubVolumeArgs{
		FS:        fs,
		SubVolume: vol,
	}
	var reply proto.KaktusCreateFsSubVolumeReply
	err := k.RPC(proto.RpcKaktusCreateFsSubVolume, args, &reply)
	if err != nil {
		return "", 0, err
	}
	return reply.Path, reply.BytesUsed, nil
}

func (k *Kylo) createNfsBackends(nfs *NFS, idStr, name, path string, clients []string) error {
	args := proto.KaktusCreateNfsBackendsArgs{
		ID:        idStr,
		Name:      name,
		FS:        nfs.FS,
		Path:      path,
		Access:    k.AccessType,
		Protocols: k.Protocols,
		Clients:   clients,
		Backends:  nfs.Ganesha.Backends,
		Port:      nfs.Ganesha.Port,
	}

	var reply proto.KaktusCreateNfsBackendsReply
	return k.RPC(proto.RpcKaktusCreateNfsBackends, args, &reply)
}

func NewKylo(projectId, regionId, nfsId, name, desc, access string, protocols []int32) (*Kylo, error) {

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

	n, err := FindNfsByID(nfsId)
	if err != nil {
		return nil, err
	}

	privateSubnetId, err := prj.GetPrivateSubnet(regionId)
	if err != nil {
		return nil, err
	}

	subnet, err := FindSubnetByID(privateSubnetId)
	if err != nil {
		return nil, err
	}

	kylo := Kylo{
		Resource:   NewResource(name, desc, MongoCollectionKyloSchemaVersion),
		ProjectID:  projectId,
		NfsID:      nfsId,
		AccessType: access,
		Protocols:  protocols,
	}

	filesystems, err := kylo.listFs()
	if err != nil {
		return nil, err
	}
	validVolume := false
	for _, fs := range filesystems {
		if n.FS == fs {
			validVolume = true
			break
		}
	}
	if !validVolume {
		return nil, fmt.Errorf("unable to find requested Ceph file system")
	}

	subvolumes, err := kylo.listSubVolumes(n.FS)
	if err != nil {
		return nil, err
	}
	for _, s := range subvolumes {
		if name == s {
			return nil, fmt.Errorf("ceph subvolume '%s' already exists", name)
		}
	}

	path, bytesUsed, err := kylo.createSubVolume(n.FS, name)
	if err != nil {
		return nil, err
	}

	exportId := n.NewExportId()
	clients := []string{subnet.CIDR}

	// create NFS export in all Ganesha backends
	err = kylo.createNfsBackends(n, exportId, name, path, clients)
	if err != nil {
		klog.Error(err)
	}

	kylo.SubVolume = KyloSubVolume{
		Name: name,
		Path: path,
		Size: bytesUsed,
	}
	kylo.ExportID = exportId
	kylo.Clients = clients

	_, err = GetDB().Insert(MongoCollectionKyloName, kylo)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new Kylo storage %s", kylo.String())

	// add Kylo to project
	prj.AddKylo(kylo.String())

	// add Kylo to NFS storage
	n.AddKylo(kylo.String(), kylo.ExportID)

	return &kylo, nil
}

func FindKylos() []Kylo {
	return FindResources[Kylo](MongoCollectionKyloName)
}

func FindKyloByProject(projectId string) ([]Kylo, error) {
	return FindResourcesByKey[Kylo](MongoCollectionKyloName, "project_id", projectId)
}

func FindKyloByNfs(nfsId string) ([]Kylo, error) {
	return FindResourcesByKey[Kylo](MongoCollectionKyloName, "nfs_id", nfsId)
}

func FindKyloByID(id string) (*Kylo, error) {
	return FindResourceByID[Kylo](MongoCollectionKyloName, id)
}

func FindKyloByName(name string) (*Kylo, error) {
	return FindResourceByName[Kylo](MongoCollectionKyloName, name)
}

func (k *Kylo) updateNfsBackends(nfs *NFS, idStr, name, path string, clients []string) error {
	args := proto.KaktusUpdateNfsBackendsArgs{
		ID:        idStr,
		Name:      name,
		FS:        nfs.FS,
		Path:      path,
		Access:    k.AccessType,
		Protocols: k.Protocols,
		Clients:   clients,
		Backends:  nfs.Ganesha.Backends,
		Port:      nfs.Ganesha.Port,
	}

	var reply proto.KaktusUpdateNfsBackendsReply
	return k.RPC(proto.RpcKaktusUpdateNfsBackends, args, &reply)
}

func (k *Kylo) Update(name, desc, access string, protocols []int32) error {
	k.UpdateResourceDefaults(name, desc)

	// find associated NFS storage
	nfs, err := k.Nfs()
	if err != nil {
		klog.Error(err)
		return err
	}

	// update values
	k.AccessType = access
	k.Protocols = protocols

	// update NFS export from all Ganesha backends
	err = k.updateNfsBackends(nfs, k.ExportID, k.SubVolume.Name, k.SubVolume.Path, k.Clients)
	if err != nil {
		klog.Error(err)
	}

	k.Save()
	return nil
}

func (k *Kylo) Project() (*Project, error) {
	return FindProjectByID(k.ProjectID)
}

func (k *Kylo) Nfs() (*NFS, error) {
	return FindNfsByID(k.NfsID)
}

func (k *Kylo) RPC(method string, args, reply any) error {
	nfs, err := k.Nfs()
	if err != nil {
		return err
	}

	return nfs.RPC(method, args, reply)
}

func (k *Kylo) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionKyloName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *Kylo) delete(fs, vol string) error {
	args := proto.KaktusDeleteFsSubVolumeArgs{
		FS:        fs,
		SubVolume: vol,
	}
	var reply proto.KaktusDeleteFsSubVolumeReply
	return k.RPC(proto.RpcKaktusDeleteFsSubVolume, args, &reply)
}

func (k *Kylo) deleteNfsBackends(nfs *NFS, idStr, name, path string, clients []string) error {
	args := proto.KaktusDeleteNfsBackendsArgs{
		ID:        idStr,
		Name:      name,
		FS:        nfs.FS,
		Path:      path,
		Access:    k.AccessType,
		Protocols: k.Protocols,
		Clients:   clients,
		Backends:  nfs.Ganesha.Backends,
		Port:      nfs.Ganesha.Port,
	}

	var reply proto.KaktusDeleteNfsBackendsReply
	return k.RPC(proto.RpcKaktusDeleteNfsBackends, args, &reply)
}

func (k *Kylo) Delete() error {
	klog.Debugf("Deleting Kylo %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}

	// find associated NFS storage
	nfs, err := k.Nfs()
	if err != nil {
		klog.Error(err)
		return err
	}

	// remove NFS export from all Ganesha backends
	err = k.deleteNfsBackends(nfs, k.ExportID, k.SubVolume.Name, k.SubVolume.Path, k.Clients)
	if err != nil {
		klog.Error(err)
	}

	// remove Ceph subvolume
	err = k.delete(nfs.FS, k.SubVolume.Name)
	if err != nil {
		return err
	}

	// remove Kylo's reference from parents
	prj, err := k.Project()
	if err != nil {
		return err
	}
	prj.RemoveKylo(k.String())
	nfs.RemoveKylo(k.String(), k.ExportID)

	return GetDB().Delete(MongoCollectionKyloName, k.ID)
}

func (k *Kylo) Model() sdk.Kylo {

	kylo := sdk.Kylo{
		Id:          k.String(),
		Name:        k.Name,
		Description: k.Description,
		Access:      k.AccessType,
		Protocols:   k.Protocols,
		Size:        k.SubVolume.Size,
	}

	nfs, err := k.Nfs()
	if err != nil {
		return kylo
	}
	kylo.Endpoint = nfs.Endpoint

	return kylo
}
