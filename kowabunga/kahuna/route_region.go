/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewRegionRouter() sdk.Router {
	return sdk.NewRegionAPIController(&RegionService{})
}

type RegionService struct{}

func (s *RegionService) CreateRegion(ctx context.Context, region sdk.Region) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("region", region))

	// check for params
	if region.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure region does not already exists
	_, err := FindRegionByName(region.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create region
	r, err := NewRegion(region.Name, region.Description)
	if err != nil {
		return HttpServerError(err)
	}

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) CreateZone(ctx context.Context, regionId string, zone sdk.Zone) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("zone", zone))

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if zone.Name == "" {
		return HttpBadParams(err)
	}

	// ensure zone does not already exists
	_, err = FindZoneByName(zone.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create zone
	z, err := NewZone(r.String(), zone.Name, zone.Description)
	if err != nil {
		return HttpServerError(err)
	}

	payload := z.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) DeleteRegion(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if region still has children referenced
	if r.HasChildren() {
		return HttpConflict(err)
	}

	// remove region
	err = r.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *RegionService) ListRegionZones(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.Zones()
	return HttpOK(payload)
}

func (s *RegionService) ListRegions(ctx context.Context) (sdk.ImplResponse, error) {
	regions := FindRegions()
	var payload []string
	for _, r := range regions {
		payload = append(payload, r.String())
	}

	return HttpOK(payload)
}

func (s *RegionService) ReadRegion(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *RegionService) UpdateRegion(ctx context.Context, regionId string, region sdk.Region) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("region", region))

	// check for params
	if region.Name == "" && region.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update region
	r.Update(region.Name, region.Description)

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *RegionService) CreateKiwi(ctx context.Context, regionId string, kiwi sdk.Kiwi) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("kiwi", kiwi))

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if kiwi.Name == "" {
		return HttpBadParams(err)
	}

	// ensure network gateway does not already exists
	_, err = FindKiwiByName(kiwi.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create network gateway
	gw, err := NewKiwi(r.String(), kiwi.Name, kiwi.Description, kiwi.Agents)
	if err != nil {
		return HttpServerError(err)
	}

	payload := gw.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) CreateVNet(ctx context.Context, regionId string, vNet sdk.VNet) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("vNet", vNet))

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if vNet.Name == "" || vNet.Interface == "" {
		return HttpBadParams(nil)
	}

	// ensure virtual network does not already exists
	_, err = FindVNetByName(vNet.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create virtual network
	v, err := NewVNet(r.String(), vNet.Name, vNet.Description, int(vNet.Vlan), vNet.Interface, vNet.Private)
	if err != nil {
		return HttpServerError(err)
	}

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) CreateStorageNFS(ctx context.Context, regionId string, storageNfs sdk.StorageNfs, poolId string) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("storageNfs", storageNfs))

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if storageNfs.Name == "" || storageNfs.Endpoint == "" || len(storageNfs.Backends) == 0 {
		return HttpBadParams(nil)
	}

	// ensure NFS Storage does not already exists
	_, err = FindNfsByName(storageNfs.Name)
	if err == nil {
		return HttpConflict(err)
	}

	fs := NfsFileSystemNameDefault // default
	if storageNfs.Fs != "" {
		fs = storageNfs.Fs
	}

	port := NfsBackendApiPort
	if storageNfs.Port != 0 {
		port = int(storageNfs.Port)
	}

	pid := r.Defaults.StoragePoolID
	if poolId != "" {
		pid = poolId
	}

	// create NFS Storage
	n, err := NewNfs(r.String(), pid, storageNfs.Name, storageNfs.Description, storageNfs.Endpoint, fs, storageNfs.Backends, port)
	if err != nil {
		return HttpServerError(err)
	}

	payload := n.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) CreateStoragePool(ctx context.Context, regionId string, storagePool sdk.StoragePool) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("regionId", regionId), RA("storagePool", storagePool))

	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if storagePool.Name == "" || storagePool.Pool == "" {
		return HttpBadParams(nil)
	}

	// ensure pool does not already exists
	_, err = FindStoragePoolByName(storagePool.Name)
	if err == nil {
		return HttpConflict(err)
	}

	var price float32
	currency := ""
	if storagePool.Cost.Price != 0 {
		price = storagePool.Cost.Price
		currency = storagePool.Cost.Currency
	}

	address := ""
	if storagePool.CephAddress != "" {
		address = storagePool.CephAddress
	}

	// create pool
	p, err := NewStoragePool(r.String(), storagePool.Name, storagePool.Description, storagePool.Pool, address, int(storagePool.CephPort), storagePool.CephSecretUuid, price, currency, storagePool.Agents)
	if err != nil {
		return HttpServerError(err)
	}

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *RegionService) ListRegionKiwis(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.Kiwis()
	return HttpOK(payload)
}

func (s *RegionService) ListRegionVNets(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.VNets()
	return HttpOK(payload)
}

func (s *RegionService) ListRegionStorageNFSs(ctx context.Context, regionId, poolId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.Nfses()
	return HttpOK(payload)
}

func (s *RegionService) ListRegionStoragePools(ctx context.Context, regionId string) (sdk.ImplResponse, error) {
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.StoragePools()
	return HttpOK(payload)
}

func (s *RegionService) SetRegionDefaultStorageNFS(ctx context.Context, regionId string, nfsId string) (sdk.ImplResponse, error) {
	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure NFS storage exists
	_, err = FindNfsByID(nfsId)
	if err != nil {
		return HttpNotFound(err)
	}

	// set default NFS storage
	err = r.SetDefaultNfs(nfsId, true)
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *RegionService) SetRegionDefaultStoragePool(ctx context.Context, regionId string, poolId string) (sdk.ImplResponse, error) {
	// ensure region exists
	r, err := FindRegionByID(regionId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure storage pool exists
	_, err = FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	// set default storage pool
	err = r.SetDefaultStoragePool(poolId, true)
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}
