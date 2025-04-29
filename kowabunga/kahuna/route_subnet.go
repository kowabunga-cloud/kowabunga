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

func NewSubnetRouter() sdk.Router {
	return sdk.NewSubnetAPIController(&SubnetService{})
}

type SubnetService struct{}

func (s *SubnetService) CreateAdapter(ctx context.Context, subnetId string, adapter sdk.Adapter, assignIP bool) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("subnetId", subnetId), RA("adapter", adapter), RA("assignIP", assignIP))

	// ensure subnet exists
	sb, err := FindSubnetByID(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if adapter.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure adapter does not already exists
	_, err = FindAdapterByName(adapter.Name)
	if err == nil {
		return HttpConflict(err)
	}

	a, err := NewAdapter(sb.String(), adapter.Name, adapter.Description, adapter.Mac, adapter.Addresses, adapter.Reserved, assignIP)
	if err != nil {
		return HttpServerError(err)
	}

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *SubnetService) DeleteSubnet(ctx context.Context, subnetId string) (sdk.ImplResponse, error) {
	// ensure subnet exists
	sb, err := FindSubnetByID(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure there's no referenced children
	if sb.HasChildren() {
		return HttpConflict(nil)
	}

	// remove subnet
	err = sb.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *SubnetService) ListSubnetAdapters(ctx context.Context, subnetId string) (sdk.ImplResponse, error) {
	sb, err := FindSubnetByID(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := sb.Adapters()
	return HttpOK(payload)
}

func (s *SubnetService) ListSubnets(ctx context.Context) (sdk.ImplResponse, error) {
	subnets := FindSubnets()
	var payload []string
	for _, s := range subnets {
		payload = append(payload, s.String())
	}

	return HttpOK(payload)
}

func (s *SubnetService) ReadSubnet(ctx context.Context, subnetId string) (sdk.ImplResponse, error) {
	sb, err := FindSubnetByID(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := sb.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *SubnetService) UpdateSubnet(ctx context.Context, subnetId string, subnet sdk.Subnet) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("subnetId", subnetId), RA("subnet", subnet))

	// check for params
	if subnet.Name == "" && subnet.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure subnet exists
	sb, err := FindSubnetByID(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update subnet
	err = sb.Update(subnet.Name, subnet.Description, subnet.Gateway, subnet.Dns, subnet.Reserved, subnet.GwPool, subnet.ExtraRoutes, subnet.Application)
	if err != nil {
		return HttpServerError(err)
	}

	payload := sb.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
