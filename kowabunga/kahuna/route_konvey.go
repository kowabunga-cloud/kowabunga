/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewKonveyRouter() sdk.Router {
	return sdk.NewKonveyAPIController(&KonveyService{})
}

type KonveyService struct{}

func (s *KonveyService) DeleteKonvey(ctx context.Context, konveyId string) (sdk.ImplResponse, error) {
	// ensure Konvey exists
	k, err := FindKonveyByID(konveyId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove Konvey
	err = k.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KonveyService) ListKonveys(ctx context.Context) (sdk.ImplResponse, error) {
	konveys := FindKonveys()
	var payload []string
	for _, k := range konveys {
		payload = append(payload, k.String())
	}

	return HttpOK(payload)
}

func (s *KonveyService) ReadKonvey(ctx context.Context, konveyId string) (sdk.ImplResponse, error) {
	k, err := FindKonveyByID(konveyId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KonveyService) UpdateKonvey(ctx context.Context, konveyId string, konvey sdk.Konvey) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("konveyId", konveyId), RA("konvey", konvey))

	// Get our konvey
	k, err := FindKonveyByID(konveyId)
	if err != nil {
		return HttpNotFound(err)
	}

	// converts from model to object
	endpoints := []KonveyEndpoint{}
	for _, ep := range konvey.Endpoints {
		e := KonveyEndpoint{
			Name:     ep.Name,
			Port:     ep.Port,
			Protocol: ep.Protocol,
			Backends: []KonveyBackend{},
		}

		err := IsValidPortListExpression(fmt.Sprintf("%d", ep.Port))
		if err != nil {
			return HttpBadParams(err)
		}

		for _, h := range ep.Backends.Hosts {
			e.Backends = append(e.Backends, KonveyBackend{
				Host: h,
				Port: ep.Backends.Port,
			})
		}

		endpoints = append(endpoints, e)
	}

	// update Konvey
	err = k.Update(k.Description, endpoints)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
