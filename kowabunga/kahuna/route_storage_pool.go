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

func NewStoragePoolRouter() sdk.Router {
	return sdk.NewPoolAPIController(&StoragePoolService{})
}

type StoragePoolService struct{}

func (s *StoragePoolService) CreateTemplate(ctx context.Context, poolId string, template sdk.Template) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("poolId", poolId), RA("template", template))

	// ensure pool exists
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if template.Name == "" {
		return HttpBadParams(err)
	}

	os := TemplateOsLinux
	if template.Os != "" {
		os = template.Os
	}

	// ensure template does not already exists in this pool
	templates, err := FindTemplatesByStoragePool(poolId)
	for _, tpl := range templates {
		if tpl.Name == template.Name {
			return HttpConflict(err)
		}
	}

	// create template
	t, err := NewTemplate(p.String(), template.Name, template.Description, os, template.Source)
	if err != nil {
		return HttpServerError(err)
	}

	payload := t.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *StoragePoolService) DeleteStoragePool(ctx context.Context, poolId string) (sdk.ImplResponse, error) {
	// ensure pool exists
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure there's no children referenced
	if p.HasChildren() {
		return HttpConflict(err)
	}

	// remove pool
	err = p.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *StoragePoolService) ListStoragePoolTemplates(ctx context.Context, poolId string) (sdk.ImplResponse, error) {
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Templates()
	return HttpOK(payload)
}

func (s *StoragePoolService) ListStoragePoolVolumes(ctx context.Context, poolId string) (sdk.ImplResponse, error) {
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Volumes()
	return HttpOK(payload)
}

func (s *StoragePoolService) ListStoragePools(ctx context.Context) (sdk.ImplResponse, error) {
	pools := FindStoragePools()
	var payload []string
	for _, p := range pools {
		payload = append(payload, p.String())
	}

	return HttpOK(payload)
}

func (s *StoragePoolService) ReadStoragePool(ctx context.Context, poolId string) (sdk.ImplResponse, error) {
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *StoragePoolService) SetStoragePoolDefaultTemplate(ctx context.Context, poolId string, templateId string) (sdk.ImplResponse, error) {
	// ensure storage pool exists
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure template exists
	_, err = p.Template(templateId)
	if err != nil {
		return HttpNotFound(err)
	}

	// set default template
	err = p.SetDefaultTemplate(templateId, true)
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *StoragePoolService) UpdateStoragePool(ctx context.Context, poolId string, storagePool sdk.StoragePool) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("poolId", poolId), RA("storagePool", storagePool))

	// check for params
	if storagePool.Name == "" && storagePool.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure pool exists
	p, err := FindStoragePoolByID(poolId)
	if err != nil {
		return HttpNotFound(err)
	}

	var price float32
	currency := ""
	if storagePool.Cost.Price != 0 {
		price = storagePool.Cost.Price
		currency = storagePool.Cost.Currency
	}

	// update pool
	p.Update(storagePool.Name, storagePool.Description, storagePool.Pool, storagePool.CephAddress, int(storagePool.CephPort), storagePool.CephSecretUuid, price, currency, storagePool.Agents)

	payload := p.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
