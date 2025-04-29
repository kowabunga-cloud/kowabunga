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

func NewVNetRouter() sdk.Router {
	return sdk.NewVnetAPIController(&VNetService{})
}

type VNetService struct{}

func (s *VNetService) CreateSubnet(ctx context.Context, vnetId string, subnet sdk.Subnet) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("vnetId", vnetId), RA("subnet", subnet))

	// ensure vnet exists
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check for params
	if subnet.Name == "" || subnet.Cidr == "" || subnet.Gateway == "" {
		return HttpBadParams(nil)
	}

	// ensure subnet does not already exists
	_, err = FindSubnetByName(subnet.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create subnet
	sb, err := NewSubnet(v.String(), subnet.Name, subnet.Description, subnet.Cidr, subnet.Gateway, subnet.Dns, v.Private, subnet.Reserved, subnet.GwPool, subnet.ExtraRoutes, subnet.Application)
	if err != nil {
		return HttpServerError(err)
	}

	payload := sb.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *VNetService) DeleteVNet(ctx context.Context, vnetId string) (sdk.ImplResponse, error) {
	// ensure vnet exists
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if vnet still has children referenced
	if v.HasChildren() {
		return HttpConflict(nil)
	}

	// remove vnet
	err = v.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *VNetService) ListVNetSubnets(ctx context.Context, vnetId string) (sdk.ImplResponse, error) {
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := v.Subnets()
	return HttpOK(payload)
}

func (s *VNetService) ListVNets(ctx context.Context) (sdk.ImplResponse, error) {
	vnets := FindVNets()
	var payload []string
	for _, v := range vnets {
		payload = append(payload, v.String())
	}

	return HttpOK(payload)
}

func (s *VNetService) ReadVNet(ctx context.Context, vnetId string) (sdk.ImplResponse, error) {
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *VNetService) SetVNetDefaultSubnet(ctx context.Context, vnetId string, subnetId string) (sdk.ImplResponse, error) {
	// ensure vnet exists
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure subnet exists
	_, err = v.Subnet(subnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// set default subnet
	err = v.SetDefaultSubnet(subnetId, true)
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *VNetService) UpdateVNet(ctx context.Context, vnetId string, vNet sdk.VNet) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("vnetId", vnetId), RA("vNet", vNet))

	// check for params
	if vNet.Name == "" && vNet.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure vnet exists
	v, err := FindVNetByID(vnetId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update vnet
	v.Update(vNet.Name, vNet.Description, int(vNet.Vlan), vNet.Interface)

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
