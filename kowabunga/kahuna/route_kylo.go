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

func NewKyloRouter() sdk.Router {
	return sdk.NewKyloAPIController(&KyloService{})
}

type KyloService struct{}

func (s *KyloService) DeleteKylo(ctx context.Context, kyloId string) (sdk.ImplResponse, error) {
	// ensure Kylo exists
	k, err := FindKyloByID(kyloId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove Kylo
	err = k.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KyloService) ListKylos(ctx context.Context) (sdk.ImplResponse, error) {
	kylos := FindKylos()
	var payload []string
	for _, k := range kylos {
		payload = append(payload, k.String())
	}

	return HttpOK(payload)
}

func (s *KyloService) ReadKylo(ctx context.Context, kyloId string) (sdk.ImplResponse, error) {
	k, err := FindKyloByID(kyloId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KyloService) UpdateKylo(ctx context.Context, kyloId string, kylo sdk.Kylo) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kyloId", kyloId), RA("kylo", kylo))

	// check for params
	if kylo.Name == "" && kylo.Access == "" {
		return HttpBadParams(nil)
	}

	// ensure Kylo exists
	k, err := FindKyloByID(kyloId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update Kylo
	err = k.Update(kylo.Name, kylo.Description, kylo.Access, kylo.Protocols)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
