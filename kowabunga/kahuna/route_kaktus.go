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

func NewKaktusRouter() sdk.Router {
	return sdk.NewKaktusAPIController(&KaktusService{})
}

type KaktusService struct{}

func (s *KaktusService) DeleteKaktus(ctx context.Context, kaktusId string) (sdk.ImplResponse, error) {
	// ensure kaktus exists
	h, err := FindKaktusByID(kaktusId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if kaktus still has children referenced
	if h.HasChildren() {
		return HttpConflict(nil)
	}

	// remove kaktus
	err = h.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KaktusService) ListKaktusInstances(ctx context.Context, kaktusId string) (sdk.ImplResponse, error) {
	h, err := FindKaktusByID(kaktusId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := h.Instances()
	return HttpOK(payload)
}

func (s *KaktusService) ListKaktuss(ctx context.Context) (sdk.ImplResponse, error) {
	kaktuss := FindKaktuses()
	var payload []string
	for _, h := range kaktuss {
		payload = append(payload, h.String())
	}

	return HttpOK(payload)
}

func (s *KaktusService) ReadKaktus(ctx context.Context, kaktusId string) (sdk.ImplResponse, error) {
	h, err := FindKaktusByID(kaktusId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := h.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KaktusService) ReadKaktusCaps(ctx context.Context, kaktusId string) (sdk.ImplResponse, error) {
	h, err := FindKaktusByID(kaktusId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := h.Capabilities.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KaktusService) UpdateKaktus(ctx context.Context, kaktusId string, kaktus sdk.Kaktus) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kaktusId", kaktusId), RA("kaktus", kaktus))

	// check for params
	if kaktus.Name == "" && kaktus.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure kaktus exists
	h, err := FindKaktusByID(kaktusId)
	if err != nil {
		return HttpNotFound(err)
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

	h.Update(kaktus.Name, kaktus.Description, cpu_price, cpu_currency, memory_price, memory_currency, overcommit_cpu, overcommit_memory, kaktus.Agents)

	payload := h.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
