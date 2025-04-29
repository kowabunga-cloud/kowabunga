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

func NewZoneRouter() sdk.Router {
	return sdk.NewZoneAPIController(&ZoneService{})
}

type ZoneService struct{}

func (s *ZoneService) CreateKaktus(ctx context.Context, zoneId string, kaktus sdk.Kaktus) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("zoneId", zoneId), RA("kaktus", kaktus))

	// ensure zone exists
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if kaktus.Name == "" {
		return HttpBadParams(err)
	}

	var cpu_price float32
	cpu_currency := CostCurrencyDefault
	if kaktus.CpuCost.Price != 0 {
		cpu_price = kaktus.CpuCost.Price
		cpu_currency = kaktus.CpuCost.Currency
	}

	var memory_price float32
	memory_currency := CostCurrencyDefault
	if kaktus.MemoryCost.Price != 0 {
		memory_price = kaktus.MemoryCost.Price
		memory_currency = kaktus.MemoryCost.Currency
	}

	var overcommit_cpu int64 = KaktusCpuOverCommmitRatio
	if kaktus.OvercommitCpuRatio != 0 {
		overcommit_cpu = kaktus.OvercommitCpuRatio
	}
	var overcommit_memory int64 = KaktusMemoryOverCommitRatio
	if kaktus.OvercommitMemoryRatio != 0 {
		overcommit_memory = kaktus.OvercommitMemoryRatio
	}

	// ensure kaktus does not already exists
	_, err = FindKaktusByName(kaktus.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create kaktus
	h, err := NewKaktus(z.String(), kaktus.Name, kaktus.Description, cpu_price, cpu_currency, memory_price, memory_currency, overcommit_cpu, overcommit_memory, kaktus.Agents)
	if err != nil {
		return HttpServerError(err)
	}

	payload := h.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *ZoneService) DeleteZone(ctx context.Context, zoneId string) (sdk.ImplResponse, error) {
	// ensure zone exists
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if zone still has children referenced
	if z.HasChildren() {
		return HttpConflict(err)
	}

	// remove zone
	err = z.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *ZoneService) ListZoneKaktuses(ctx context.Context, zoneId string) (sdk.ImplResponse, error) {
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := z.Kaktuses()
	return HttpOK(payload)
}

func (s *ZoneService) ListZones(ctx context.Context) (sdk.ImplResponse, error) {
	zones := FindZones()
	var payload []string
	for _, z := range zones {
		payload = append(payload, z.String())
	}

	return HttpOK(payload)
}

func (s *ZoneService) ReadZone(ctx context.Context, zoneId string) (sdk.ImplResponse, error) {
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := z.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *ZoneService) UpdateZone(ctx context.Context, zoneId string, zone sdk.Zone) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("zoneId", zoneId), RA("zone", zone))

	// check for params
	if zone.Name == "" && zone.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure zone exists
	z, err := FindZoneByID(zoneId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update zone
	z.Update(zone.Name, zone.Description)

	payload := z.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
