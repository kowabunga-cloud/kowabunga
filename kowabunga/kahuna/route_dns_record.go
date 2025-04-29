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

func NewDnsRecordRouter() sdk.Router {
	return sdk.NewRecordAPIController(&DnsRecordService{})
}

type DnsRecordService struct{}

func (s *DnsRecordService) DeleteDnsRecord(ctx context.Context, recordId string) (sdk.ImplResponse, error) {
	// ensure record exists
	r, err := FindDnsRecordByID(recordId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove record
	err = r.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *DnsRecordService) ReadDnsRecord(ctx context.Context, recordId string) (sdk.ImplResponse, error) {
	r, err := FindDnsRecordByID(recordId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *DnsRecordService) UpdateDnsRecord(ctx context.Context, recordId string, dnsRecord sdk.DnsRecord) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("recordId", recordId), RA("dnsRecord", dnsRecord))

	// check for params
	if dnsRecord.Name == "" && len(dnsRecord.Addresses) == 0 {
		return HttpBadParams(nil)
	}

	// ensure record exists
	r, err := FindDnsRecordByID(recordId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update record
	r.Update(dnsRecord.Name, dnsRecord.Description, dnsRecord.Addresses)

	payload := r.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
