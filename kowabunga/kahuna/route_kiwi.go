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

func NewKiwiRouter() sdk.Router {
	return sdk.NewKiwiAPIController(&KiwiService{})
}

type KiwiService struct{}

func (s *KiwiService) DeleteKiwi(ctx context.Context, kiwiId string) (sdk.ImplResponse, error) {
	// ensure kiwi exists
	gw, err := FindKiwiByID(kiwiId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove kiwi
	err = gw.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KiwiService) ListKiwis(ctx context.Context) (sdk.ImplResponse, error) {
	kiwis := FindKiwis()
	var payload []string
	for _, gw := range kiwis {
		payload = append(payload, gw.String())
	}

	return HttpOK(payload)
}

func (s *KiwiService) ReadKiwi(ctx context.Context, kiwiId string) (sdk.ImplResponse, error) {
	gw, err := FindKiwiByID(kiwiId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := gw.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KiwiService) UpdateKiwi(ctx context.Context, kiwiId string, kiwi sdk.Kiwi) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kiwiId", kiwiId), RA("kiwi", kiwi))

	// check for params
	if kiwi.Name == "" && kiwi.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure kiwi exists
	gw, err := FindKiwiByID(kiwiId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update kiwi
	gw.Update(kiwi.Name, kiwi.Description, kiwi.Agents)

	payload := gw.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
