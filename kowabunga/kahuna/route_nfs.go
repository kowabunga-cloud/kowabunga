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

func NewNfsRouter() sdk.Router {
	return sdk.NewNfsAPIController(&NfsService{})
}

type NfsService struct{}

func (s *NfsService) DeleteStorageNFS(ctx context.Context, nfsId string) (sdk.ImplResponse, error) {
	// ensure NFS storage exists
	n, err := FindNfsByID(nfsId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure there's no children referenced
	if n.HasChildren() {
		return HttpConflict(nil)
	}

	// remove NFS storage
	err = n.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *NfsService) ListStorageNFSKylos(ctx context.Context, nfsId string) (sdk.ImplResponse, error) {
	n, err := FindNfsByID(nfsId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := n.Kylos()
	return HttpOK(payload)
}

func (s *NfsService) ListStorageNFSs(ctx context.Context) (sdk.ImplResponse, error) {
	storages := FindNFSes()
	var payload []string
	for _, s := range storages {
		payload = append(payload, s.String())
	}

	return HttpOK(payload)
}

func (s *NfsService) ReadStorageNFS(ctx context.Context, nfsId string) (sdk.ImplResponse, error) {
	n, err := FindNfsByID(nfsId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := n.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *NfsService) UpdateStorageNFS(ctx context.Context, nfsId string, storageNfs sdk.StorageNfs) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("nfsId", nfsId), RA("storageNfs", storageNfs))

	// check for params
	if storageNfs.Name == "" && storageNfs.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure NFS storage exists
	n, err := FindNfsByID(nfsId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update NFS storage
	n.Update(storageNfs.Name, storageNfs.Description, storageNfs.Endpoint, storageNfs.Fs, storageNfs.Backends, int(storageNfs.Port))

	payload := n.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
