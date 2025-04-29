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

func NewAdapterRouter() sdk.Router {
	return sdk.NewAdapterAPIController(&AdapterService{})
}

type AdapterService struct{}

func (s *AdapterService) DeleteAdapter(ctx context.Context, adapterId string) (sdk.ImplResponse, error) {
	// ensure adapter exists
	a, err := FindAdapterByID(adapterId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove adapter
	err = a.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *AdapterService) ListAdapters(ctx context.Context) (sdk.ImplResponse, error) {
	adapters := FindAdapters()
	var payload []string
	for _, a := range adapters {
		payload = append(payload, a.String())
	}

	return HttpOK(payload)
}

func (s *AdapterService) ReadAdapter(ctx context.Context, adapterId string) (sdk.ImplResponse, error) {
	a, err := FindAdapterByID(adapterId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *AdapterService) UpdateAdapter(ctx context.Context, adapterId string, adapter sdk.Adapter) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("adapterId", adapterId), RA("adapter", adapter))

	// check for params
	if adapter.Name == "" && adapter.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure adapter exists
	a, err := FindAdapterByID(adapterId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update adapter
	a.Update(adapter.Name, adapter.Description, adapter.Mac, adapter.Addresses, adapter.Reserved)

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
