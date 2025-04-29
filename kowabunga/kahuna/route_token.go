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

func NewTokenRouter() sdk.Router {
	return sdk.NewTokenAPIController(&TokenService{})
}

type TokenService struct{}

func (s *TokenService) DeleteApiToken(ctx context.Context, tokenId string) (sdk.ImplResponse, error) {
	// ensure token exists
	t, err := FindTokenByID(tokenId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove token
	err = t.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *TokenService) ListApiTokens(ctx context.Context) (sdk.ImplResponse, error) {
	tokens := FindTokens()
	var payload []string
	for _, t := range tokens {
		payload = append(payload, t.String())
	}

	return HttpOK(payload)
}

func (s *TokenService) ReadApiToken(ctx context.Context, tokenId string) (sdk.ImplResponse, error) {
	t, err := FindTokenByID(tokenId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := t.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *TokenService) UpdateApiToken(ctx context.Context, tokenId string, apiToken sdk.ApiToken) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("tokenId", tokenId), RA("apiToken", apiToken))

	// check for params
	if apiToken.Name == "" && apiToken.Description == "" && apiToken.ExpirationDate == "" {
		return HttpBadParams(nil)
	}

	// ensure token exists
	t, err := FindTokenByID(tokenId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update token
	err = t.Update(apiToken.Name, apiToken.Description, apiToken.Expire, apiToken.ExpirationDate)
	if err != nil {
		return HttpServerError(err)
	}

	payload := t.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
