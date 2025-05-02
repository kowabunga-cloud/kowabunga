/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/joeig/go-powerdns/v3"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	PowerDnsDefaultHost             = "localhost"
	PowerDnsNameServer              = "localhost."
	PowerDnsHeaderAPI               = "X-API-Key"
	PowerDnsTTL                     = 3600
	PowerDnsRecursorZoneType        = "Zone"
	PowerDnsRecursorZoneKind        = "Forwarded"
	PowerDnsRecursorZoneApiEndpoint = "/api/v1/servers/localhost/zones"

	PowerDnsErrorServer             = "unable to process request"
	PowerDnsErrorNoZone             = "no such zone"
	PowerDnsErrorZoneNotEmpty       = "zone has existing records"
	PowerDnsErrorNoRecord           = "no such record"
	PowerDnsErrorWrongTypeRecord    = "already existing record with non-A type"
	PowerDnsErrorRecursorZoneCreate = "unable to create zone within recursor: [%d] %s"
	PowerDnsErrorRecursorZoneDelete = "unable to delete zone within recursor: [%d] %s"
)

func pdnsError(reason string, e error) error {
	err := fmt.Errorf("[PDNS] %s %#v", reason, e)
	klog.Error(err)
	return err
}

func pdnsZone(domain string) string {
	if strings.HasSuffix(domain, ".") {
		return domain
	}
	return domain + "."
}

// PowerDNS Authoritative Service

type PowerDnsAuthoritiveServer struct {
	Client *powerdns.Client
}

func (as *PowerDnsAuthoritiveServer) GetZone(domain string) (*powerdns.Zone, error) {
	zone := pdnsZone(domain)
	return as.Client.Zones.Get(context.Background(), zone)
}

func (as *PowerDnsAuthoritiveServer) CreateZone(domain string) (*powerdns.Zone, error) {
	zone := pdnsZone(domain)
	return as.Client.Zones.AddNative(context.Background(), zone, false, "", false,
		"", "", true, []string{PowerDnsNameServer})
}

func (as *PowerDnsAuthoritiveServer) DeleteZone(domain string) error {
	zone := pdnsZone(domain)
	return as.Client.Zones.Delete(context.Background(), zone)
}

func (as *PowerDnsAuthoritiveServer) GetZoneRecord(zone *powerdns.Zone, entry string) (*powerdns.RRset, error) {
	if zone == nil {
		return nil, fmt.Errorf("%s", PowerDnsErrorNoZone)
	}

	for _, r := range zone.RRsets {
		record := fmt.Sprintf("%s.%s", entry, *zone.ID)
		if *r.Name == record {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("%s", PowerDnsErrorNoRecord)
}

func (as *PowerDnsAuthoritiveServer) CreateZoneRecord(domain, entry string, addr []string) error {
	zone := pdnsZone(domain)
	record := fmt.Sprintf("%s.%s", entry, domain)
	return as.Client.Records.Add(context.Background(), zone, record, powerdns.RRTypeA, PowerDnsTTL, addr)
}

func (as *PowerDnsAuthoritiveServer) UpdateZoneRecord(domain, entry string, addr []string) error {
	zone := pdnsZone(domain)
	record := fmt.Sprintf("%s.%s", entry, domain)
	return as.Client.Records.Change(context.Background(), zone, record, powerdns.RRTypeA, PowerDnsTTL, addr)
}

func (as *PowerDnsAuthoritiveServer) DeleteZoneRecord(domain, entry string) error {
	zone := pdnsZone(domain)
	record := fmt.Sprintf("%s.%s", entry, domain)
	return as.Client.Records.Delete(context.Background(), zone, record, powerdns.RRTypeA)
}

func newPowerDnsAuthoritativeServer(cfg KiwiAgentPowerDnsConfig) *PowerDnsAuthoritiveServer {
	uri := fmt.Sprintf("http://%s:%d", cfg.AS.Host, cfg.AS.APIPort)
	client := powerdns.New(uri, PowerDnsDefaultHost, powerdns.WithAPIKey(cfg.AS.APIKey))

	return &PowerDnsAuthoritiveServer{
		Client: client,
	}
}

// PowerDNS Recursor Service

type PowerDnsRecursor struct {
	Host    string
	Port    int
	BaseURI string
	Headers map[string]string
	Client  *http.Client
}

type PowerDnsRecursorApiZone struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Kind      string   `json:"kind"`
	Recursion bool     `json:"recursion_desired"`
	Servers   []string `json:"servers"`
}

func (pr *PowerDnsRecursor) request(method, path string, body interface{}) (*http.Request, error) {
	var b io.Reader = nil
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		b = bytes.NewReader(buf)
	}

	uri := pr.BaseURI + path
	req, err := http.NewRequest(method, uri, b)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", common.MimeJSON)
		req.Header.Set("Accept", common.MimeJSON)
	}
	req.Header.Set("User-Agent", KiwiAppNmame)
	for k, v := range pr.Headers {
		req.Header.Set(k, v)
	}

	klog.Debugf("Issuing PowerDNS HTTP request %#v", req)
	return req, nil
}

func (pr *PowerDnsRecursor) CreateZone(domain string, servers []string) error {
	zone := pdnsZone(domain)
	z := PowerDnsRecursorApiZone{
		Name:      zone,
		Type:      PowerDnsRecursorZoneType,
		Kind:      PowerDnsRecursorZoneKind,
		Recursion: false,
		Servers:   servers,
	}

	req, err := pr.request("POST", PowerDnsRecursorZoneApiEndpoint, z)
	if err != nil {
		return err
	}

	resp, err := pr.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 201 && resp.StatusCode != 422 {
		return fmt.Errorf(PowerDnsErrorRecursorZoneCreate, resp.StatusCode, resp.Status)
	}

	return nil
}

func (pr *PowerDnsRecursor) DeleteZone(domain string) error {
	path := PowerDnsRecursorZoneApiEndpoint + "/" + domain
	req, err := pr.request("DELETE", path, nil)
	if err != nil {
		return err
	}

	resp, err := pr.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 204 {
		return fmt.Errorf(PowerDnsErrorRecursorZoneDelete, resp.StatusCode, resp.Status)
	}

	return nil
}

func newPowerDnsRecursor(cfg KiwiAgentPowerDnsRecursorConfig) *PowerDnsRecursor {
	return &PowerDnsRecursor{
		Host:    cfg.Host,
		Port:    cfg.Port,
		BaseURI: fmt.Sprintf("http://%s:%d", cfg.Host, cfg.APIPort),
		Headers: map[string]string{
			PowerDnsHeaderAPI: cfg.APIKey,
		},
		Client: http.DefaultClient,
	}
}

func newPowerDnsRecursorServers(cfg KiwiAgentPowerDnsConfig) []*PowerDnsRecursor {
	recursors := []*PowerDnsRecursor{}
	for _, r := range cfg.Recursors {
		rec := newPowerDnsRecursor(r)
		recursors = append(recursors, rec)
	}
	return recursors
}

// Public API

type PowerDnsConnectionSettings struct {
	AS        *PowerDnsAuthoritiveServer
	Recursors []*PowerDnsRecursor
	Endpoints []string
}

func NewPowerDnsConnectionSettings(cfg *KiwiAgentConfig) (*PowerDnsConnectionSettings, error) {
	pdns := PowerDnsConnectionSettings{
		AS:        newPowerDnsAuthoritativeServer(cfg.PowerDNS),
		Recursors: newPowerDnsRecursorServers(cfg.PowerDNS),
		Endpoints: []string{},
	}

	for _, r := range pdns.Recursors {
		e := fmt.Sprintf("%s:%d", r.Host, r.Port)
		pdns.Endpoints = append(pdns.Endpoints, e)
	}

	klog.Infof("Successfully initiated PowerDNS connection")
	return &pdns, nil
}

func (pdns *PowerDnsConnectionSettings) CreateDnsZone(domain string) error {
	klog.Infof("Creating DNS zone %s ...", domain)

	// check if zone does not already exists
	_, err := pdns.AS.GetZone(domain)
	if err == nil {
		// it does, good, nothing more to do here !
		return nil
	}
	if err != nil {
		_, ok := err.(*powerdns.Error)
		if ok {
			if err.(*powerdns.Error).StatusCode == 409 {
				return nil
			}
		}
	}

	// otherwise, let's create a new zone within authoritative server
	_, err = pdns.AS.CreateZone(domain)
	if err != nil {
		return pdnsError(PowerDnsErrorServer, err)
	}

	// loop over recursors to declare newly created zone
	for _, r := range pdns.Recursors {
		err := r.CreateZone(domain, pdns.Endpoints)
		if err != nil {
			return pdnsError(PowerDnsErrorServer, err)
		}
	}

	return nil
}

func (pdns *PowerDnsConnectionSettings) DeleteDnsZone(domain string) error {
	klog.Infof("Deleting DNS zone %s ...", domain)

	// ensure zone exists
	z, _ := pdns.AS.GetZone(domain)
	if z == nil {
		// zone doesn't exist, good, nothing more to do here !
		return nil
	}

	// ensure zone has no other record but SOA and NS top-level entries
	if len(z.RRsets) > 2 {
		return pdnsError(PowerDnsErrorZoneNotEmpty, nil)
	}

	// remove zone from authoritative server
	err := pdns.AS.DeleteZone(domain)
	if err != nil {
		klog.Errorf("Can't remove zone %s for AS: %#v", domain, err)
		return pdnsError(PowerDnsErrorServer, err)
	}

	// loop over recursors to remove zone
	for _, r := range pdns.Recursors {
		err = r.DeleteZone(domain)
		if err != nil {
			klog.Errorf("Can't remove zone %s for recursor %s: %#v", domain, r.Host, err)
			return pdnsError(PowerDnsErrorServer, err)
		}
	}

	return nil
}

func (pdns *PowerDnsConnectionSettings) CreateDnsRecord(domain, entry string, addr []string) error {
	klog.Infof("Creating DNS record for %s.%s ...", entry, domain)

	// ensure zone exists
	z, err := pdns.AS.GetZone(domain)
	if err != nil {
		return pdnsError(PowerDnsErrorNoZone, err)
	}

	// check if record already exists
	sort.Strings(addr)
	zr, err := pdns.AS.GetZoneRecord(z, entry)
	if zr != nil {
		// it does, but with wrong type, that's a problem ...
		if *zr.Type != powerdns.RRTypeA {
			return pdnsError(PowerDnsErrorWrongTypeRecord, err)
		}

		// it does, and addresses are all the same, good, nothing more to do here !
		addresses := []string{}
		for _, r := range zr.Records {
			addresses = append(addresses, *r.Content)
		}
		sort.Strings(addresses)
		if reflect.DeepEqual(addresses, addr) {
			return nil
		}
	}

	// otherwise, let's create the new record
	err = pdns.AS.CreateZoneRecord(domain, entry, addr)
	if err != nil {
		return pdnsError(PowerDnsErrorServer, err)
	}

	return nil
}

func (pdns *PowerDnsConnectionSettings) UpdateDnsRecord(domain, entry string, addr []string) error {
	klog.Infof("Updating DNS record for %s.%s ...", entry, domain)

	// ensure zone exists
	z, err := pdns.AS.GetZone(domain)
	if err != nil {
		return pdnsError(PowerDnsErrorNoZone, err)
	}

	// ensure record exists
	_, err = pdns.AS.GetZoneRecord(z, entry)
	if err != nil {
		return pdnsError(PowerDnsErrorNoRecord, err)
	}

	// otherwise, update the record
	err = pdns.AS.UpdateZoneRecord(domain, entry, addr)
	if err != nil {
		return pdnsError(PowerDnsErrorServer, err)
	}

	return nil
}

func (pdns *PowerDnsConnectionSettings) DeleteDnsRecord(domain, entry string) error {
	klog.Infof("Deleting DNS record for %s.%s ...", entry, domain)

	// ensure zone exists
	z, err := pdns.AS.GetZone(domain)
	if err != nil {
		return pdnsError(PowerDnsErrorNoZone, err)
	}

	// ensure record exists
	rs, _ := pdns.AS.GetZoneRecord(z, entry)
	if rs == nil {
		// record doesn't exist, good, nothing more to do here !
		return nil
	}

	// otherwise, drop the record
	err = pdns.AS.DeleteZoneRecord(domain, entry)
	if err != nil {
		return pdnsError(PowerDnsErrorServer, err)
	}

	return nil
}
