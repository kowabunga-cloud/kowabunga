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

func NewKawaiiRouter() sdk.Router {
	return sdk.NewKawaiiAPIController(&KawaiiService{})
}

type KawaiiService struct{}

func (s *KawaiiService) DeleteKawaii(ctx context.Context, kawaiiId string) (sdk.ImplResponse, error) {
	// ensure Kawaii exists
	k, err := FindKawaiiByID(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}

	if k.HasChildren() {
		return HttpConflict(nil)
	}
	// remove Kawaii
	err = k.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KawaiiService) ListKawaiis(ctx context.Context) (sdk.ImplResponse, error) {
	kawaiis := FindKawaiis()
	var payload []string
	for _, k := range kawaiis {
		payload = append(payload, k.String())
	}

	return HttpOK(payload)
}

func (s *KawaiiService) ReadKawaii(ctx context.Context, kawaiiId string) (sdk.ImplResponse, error) {
	k, err := FindKawaiiByID(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KawaiiService) UpdateKawaii(ctx context.Context, kawaiiId string, kawaii sdk.Kawaii) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kawaiiId", kawaiiId), RA("kawaii", kawaii))

	// Get our kawaii
	gw, err := FindKawaiiByID(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}

	// converts kawaii from model to object
	fw := KawaiiFirewall{
		EgressPolicy: kawaii.Firewall.EgressPolicy,
	}
	if fw.EgressPolicy == "" {
		fw.EgressPolicy = KawaiiFirewallPolicyAccept
	}
	for _, in := range kawaii.Firewall.Ingress {
		err := IsValidPortListExpression(in.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallIngressRule{
			Source:   in.Source,
			Protocol: in.Protocol,
			Ports:    in.Ports,
		}
		if rule.Source == "" {
			rule.Source = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Ingress = append(fw.Ingress, rule)
	}
	for _, out := range kawaii.Firewall.Egress {
		err := IsValidPortListExpression(out.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallEgressRule{
			Destination: out.Destination,
			Protocol:    out.Protocol,
			Ports:       out.Ports,
		}
		if rule.Destination == "" {
			rule.Destination = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Egress = append(fw.Egress, rule)
	}

	natRules := []KawaiiDNatRule{}
	for _, rule := range kawaii.Dnat {
		err := IsValidPortListExpression(rule.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiDNatRule{
			PrivateIP: rule.Destination,
			Protocol:  rule.Protocol,
			Ports:     rule.Ports,
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		natRules = append(natRules, rule)
	}

	// update Kawaii
	err = gw.Update(kawaii.Description, fw, natRules)
	if err != nil {
		return HttpServerError(err)
	}

	payload := gw.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

// IPSEC Connection
func (s *KawaiiService) CreateKawaiiIpSec(ctx context.Context, kawaiiId string, ipsec sdk.KawaiiIpSec) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kawaiiId", kawaiiId), RA("ipSecConnection", ipsec))

	// Ensure kawaii exists
	_, err := FindKawaiiByID(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}
	existingIpsecs, err := FindIPsecByKawaii(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}
	for _, existingIpsec := range existingIpsecs {
		if existingIpsec.Name == ipsec.Name || existingIpsec.RemotePeer == ipsec.RemoteIp {
			return HttpConflict(fmt.Errorf("IP Sec connection (name and remotePeer) must be unique : %s/%s", existingIpsec.Name, existingIpsec.RemotePeer))
		}
	}
	fw := KawaiiFirewall{}
	for _, in := range ipsec.Firewall.Ingress {
		err := IsValidPortListExpression(in.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallIngressRule{
			Source:   in.Source,
			Protocol: in.Protocol,
			Ports:    in.Ports,
		}
		if rule.Source == "" {
			rule.Source = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Ingress = append(fw.Ingress, rule)
	}
	ipsecConn, err := NewKawaiiIPsecConnection(kawaiiId, ipsec.Name, ipsec.Description, ipsec.RemoteIp,
		ipsec.RemoteSubnet, ipsec.PreSharedKey,
		ipsec.DpdTimeout, ipsec.DpdTimeoutAction, ipsec.StartAction, ipsec.RekeyTime,
		ipsec.Phase1Lifetime, ipsec.Phase1DhGroupNumber, ipsec.Phase1IntegrityAlgorithm, ipsec.Phase1EncryptionAlgorithm,
		ipsec.Phase2Lifetime, ipsec.Phase2DhGroupNumber, ipsec.Phase2IntegrityAlgorithm, ipsec.Phase2EncryptionAlgorithm,
		fw)
	if err != nil {
		return HttpServerError(err)
	}
	payload := ipsecConn.Model()
	LogHttpResponse(hideSecretsPayload(payload))
	return HttpCreated(payload)
}

func (s *KawaiiService) DeleteKawaiiIpSec(ctx context.Context, kawaiiId, ipsecId string) (sdk.ImplResponse, error) {
	// ensure Kawaii exists
	k, err := FindIPsecByID(ipsecId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove Kawaii
	err = k.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KawaiiService) ListKawaiiIpSecs(ctx context.Context, kawaiiId string) (sdk.ImplResponse, error) {
	ipsecs, err := FindIPsecByKawaii(kawaiiId)
	if err != nil {
		return HttpNotFound(err)
	}
	var payload []string
	for _, k := range ipsecs {
		payload = append(payload, k.String())
	}

	return HttpOK(payload)
}

func (s *KawaiiService) ReadKawaiiIpSec(ctx context.Context, kawaiiId, ipsecId string) (sdk.ImplResponse, error) {
	// Ensure Ip Sec Conn exists
	ipsec, err := FindIPsecByID(ipsecId)
	if err != nil {
		return HttpNotFound(err)
	}
	payload := ipsec.Model()
	LogHttpResponse(hideSecretsPayload(payload))
	return HttpOK(payload)
}

func (s *KawaiiService) UpdateKawaiiIpSec(ctx context.Context, kawaiiId, ipsecId string, ipsecSpec sdk.KawaiiIpSec) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("kawaiiId", kawaiiId), RA("ipsec", ipsecId))

	// Ensure Ip Sec Conn exists
	ipsecConn, err := FindIPsecByID(ipsecId)
	if err != nil {
		return HttpNotFound(err)
	}
	fw := KawaiiFirewall{}
	for _, in := range ipsecSpec.Firewall.Ingress {
		err := IsValidPortListExpression(in.Ports)
		if err != nil {
			return HttpBadParams(err)
		}

		rule := KawaiiFirewallIngressRule{
			Source:   in.Source,
			Protocol: in.Protocol,
			Ports:    in.Ports,
		}
		if rule.Source == "" {
			rule.Source = KawaiiFirewallWildcardNetwork
		}
		if rule.Protocol == "" {
			rule.Protocol = KawaiiFirewallProtocolTCP
		}
		fw.Ingress = append(fw.Ingress, rule)
	}
	err = ipsecConn.Update(ipsecSpec.Name, ipsecSpec.Description, ipsecSpec.RemoteIp,
		ipsecSpec.RemoteSubnet, ipsecSpec.PreSharedKey,
		ipsecSpec.DpdTimeout, ipsecSpec.DpdTimeoutAction, ipsecSpec.StartAction, ipsecSpec.RekeyTime,
		ipsecSpec.Phase1Lifetime, ipsecSpec.Phase1DhGroupNumber, ipsecSpec.Phase1IntegrityAlgorithm, ipsecSpec.Phase1EncryptionAlgorithm,
		ipsecSpec.Phase2Lifetime, ipsecSpec.Phase2DhGroupNumber, ipsecSpec.Phase2IntegrityAlgorithm, ipsecSpec.Phase2EncryptionAlgorithm,
		fw)
	if err != nil {
		return HttpServerError(err)
	}
	payload := ipsecConn.Model()
	LogHttpResponse(hideSecretsPayload(payload))
	return HttpOK(payload)
}

func hideSecretsPayload(payload sdk.KawaiiIpSec) sdk.KawaiiIpSec {
	payload.PreSharedKey = "<redacted>"
	return payload
}
