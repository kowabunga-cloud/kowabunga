/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
	"golang.org/x/exp/rand"
)

const (
	MongoCollectionIPsecName                = "ipsec"
	MongoCollectionKawaiiIPsecSchemaVersion = 1
	ErrorIPsecUnderlyingMZRNotFound         = "Could not find kawaii underlying MZR and consequently public IPs to assign to the IPSEC"
)

type KawaiiIPsec struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	KawaiiID string `bson:"kawaii_id"`

	// properties
	IP                        string         `bson:"ip"`
	XfrmID                    uint8          `bson:"xfrm_id"`
	RemotePeer                string         `bson:"remote_peer"`
	RemoteSubnet              string         `bson:"remote_subnet"`
	PreSharedKey              string         `bson:"pre_shared_key"`
	DpdTimeout                string         `bson:"dpd_timeout"`
	DpdTimeoutAction          string         `bson:"dpd_action"`
	StartAction               string         `bson:"start_action"`
	Rekey                     string         `bson:"rekey"`
	Phase1Lifetime            string         `bson:"phase1_lifetime"`
	Phase1DHGroupNumber       int64          `bson:"phase1_df_group"`
	Phase1IntegrityAlgorithm  string         `bson:"phase1_integrity_algorithm"`
	Phase1EncryptionAlgorithm string         `bson:"phase1_encryption_algorithm"`
	Phase2Lifetime            string         `bson:"phase2_lifetime"`
	Phase2DHGroupNumber       int64          `bson:"phase2_df_group"`
	Phase2IntegrityAlgorithm  string         `bson:"phase2_integrity_algorithm"`
	Phase2EncryptionAlgorithm string         `bson:"phase2_encryption_algorithm"`
	Firewall                  KawaiiFirewall `bson:"firewall"`
}

func NewKawaiiIPsecConnection(kawaiiId, name, desc, remotePeer, remoteSubnet, preSharedKey string,
	dpdTimeout, dpdTimeoutAction, startAction, rekey string,
	p1Lifetime string, p1DHGroupNumber int64, p1IntegrityAlgorithm, p1EncryptionAlgorithm string,
	p2Lifetime string, p2DHGroupNumber int64, p2IntegrityAlgorithm, p2EncryptionAlgorithm string,
	fw KawaiiFirewall) (*KawaiiIPsec, error) {
	kawaii, err := FindKawaiiByID(kawaiiId)
	if err != nil {
		return nil, err
	}
	mzr, err := kawaii.MZR()
	if err != nil {
		return nil, fmt.Errorf("%s", ErrorIPsecUnderlyingMZRNotFound)
	}
	ipSecIP := mzr.PublicVIPs[rand.Intn(len(mzr.PublicVIPs))]

	ipsecs, err := FindIPsecByKawaii(kawaiiId)
	if err != nil {
		return nil, err
	}

	var xfrmId uint8 = 1
	xfrmIdfound := true
	for xfrmIdfound {
		xfrmIdfound = false
		for _, ipsec := range ipsecs {
			if ipsec.XfrmID == xfrmId {
				xfrmIdfound = true
				xfrmId++
				break
			}
		}
	}

	kawaiiIPsec := KawaiiIPsec{
		Resource:                  NewResource(name, desc, MongoCollectionKawaiiIPsecSchemaVersion),
		KawaiiID:                  kawaiiId,
		IP:                        ipSecIP,
		XfrmID:                    xfrmId,
		RemotePeer:                remotePeer,
		RemoteSubnet:              remoteSubnet,
		PreSharedKey:              preSharedKey,
		DpdTimeout:                dpdTimeout,
		DpdTimeoutAction:          dpdTimeoutAction,
		StartAction:               startAction,
		Rekey:                     rekey,
		Phase1Lifetime:            p1Lifetime,
		Phase1DHGroupNumber:       p1DHGroupNumber,
		Phase1IntegrityAlgorithm:  p1IntegrityAlgorithm,
		Phase1EncryptionAlgorithm: p1EncryptionAlgorithm,
		Phase2Lifetime:            p2Lifetime,
		Phase2DHGroupNumber:       p2DHGroupNumber,
		Phase2IntegrityAlgorithm:  p2IntegrityAlgorithm,
		Phase2EncryptionAlgorithm: p2EncryptionAlgorithm,
		Firewall:                  fw,
	}

	klog.Debugf("Created new Kawaii IPsec %s", kawaiiIPsec.String())
	_, err = GetDB().Insert(MongoCollectionIPsecName, kawaiiIPsec)
	if err != nil {
		return nil, err
	}

	// Ref inside kawaii + IP Sec ref to Kawaii ?
	kawaii.AddIPsec(kawaiiIPsec.String())

	return &kawaiiIPsec, nil
}

func (k *KawaiiIPsec) Model() sdk.KawaiiIpSec {
	return sdk.KawaiiIpSec{
		Id:                        k.String(),
		Description:               k.Description,
		Ip:                        k.IP,
		Name:                      k.Name,
		RemoteIp:                  k.RemotePeer,
		RemoteSubnet:              k.RemoteSubnet,
		PreSharedKey:              k.PreSharedKey,
		DpdTimeout:                k.DpdTimeout,
		DpdTimeoutAction:          k.DpdTimeoutAction,
		StartAction:               k.StartAction,
		RekeyTime:                 k.Rekey,
		Phase1Lifetime:            k.Phase1Lifetime,
		Phase1DhGroupNumber:       k.Phase1DHGroupNumber,
		Phase1IntegrityAlgorithm:  k.Phase1IntegrityAlgorithm,
		Phase1EncryptionAlgorithm: k.Phase1EncryptionAlgorithm,
		Phase2Lifetime:            k.Phase2Lifetime,
		Phase2DhGroupNumber:       k.Phase2DHGroupNumber,
		Phase2IntegrityAlgorithm:  k.Phase2IntegrityAlgorithm,
		Phase2EncryptionAlgorithm: k.Phase2EncryptionAlgorithm,
		Firewall:                  k.Firewall.Model(),
	}
}

func FindKawaiisIPsec() []KawaiiIPsec {
	return FindResources[KawaiiIPsec](MongoCollectionIPsecName)
}

func FindIPsecByKawaii(kawaiiId string) ([]KawaiiIPsec, error) {
	return FindResourcesByKey[KawaiiIPsec](MongoCollectionIPsecName, "kawaii_id", kawaiiId)
}

func FindIPsecByID(id string) (*KawaiiIPsec, error) {
	return FindResourceByID[KawaiiIPsec](MongoCollectionIPsecName, id)
}

func (k *KawaiiIPsec) Save() {
	k.Updated()
	_, err := GetDB().Update(MongoCollectionIPsecName, k.ID, k)
	if err != nil {
		klog.Error(err)
	}
}

func (k *KawaiiIPsec) Update(name, desc, remotePeer, remoteSubnet, preSharedKey string,
	dpdTimeout, dpdTimeoutAction, startAction, rekey string,
	p1Lifetime string, p1DHGroupNumber int64, p1IntegrityAlgorithm, p1EncryptionAlgorithm string,
	p2Lifetime string, p2DHGroupNumber int64, p2IntegrityAlgorithm, p2EncryptionAlgorithm string,
	fw KawaiiFirewall) error {
	k.Name = name
	k.PreSharedKey = preSharedKey
	k.Description = desc
	k.Firewall = fw
	k.RemotePeer = remotePeer
	k.RemoteSubnet = remoteSubnet
	k.DpdTimeout = dpdTimeout
	k.DpdTimeoutAction = dpdTimeoutAction
	k.StartAction = startAction
	k.Rekey = rekey
	k.Phase1Lifetime = p1Lifetime
	k.Phase1DHGroupNumber = p1DHGroupNumber
	k.Phase1IntegrityAlgorithm = p1IntegrityAlgorithm
	k.Phase1EncryptionAlgorithm = p1EncryptionAlgorithm
	k.Phase2Lifetime = p2Lifetime
	k.Phase2DHGroupNumber = p2DHGroupNumber
	k.Phase2IntegrityAlgorithm = p2IntegrityAlgorithm
	k.Phase2EncryptionAlgorithm = p2EncryptionAlgorithm
	k.Save()

	parentKawaii, err := FindKawaiiByID(k.KawaiiID)
	if err != nil {
		return err
	}
	mzr, err := parentKawaii.MZR()
	if err != nil {
		return nil // bypass error
	}

	for _, komputeId := range mzr.KomputeIDs {
		kompute, err := FindKomputeByID(komputeId)
		if err != nil {
			continue
		}

		i, err := kompute.Instance()
		if err != nil {
			continue
		}

		args := agents.KontrollerReloadArgs{}
		var reply agents.KontrollerReloadReply
		err = i.InstanceRPC("Reload", args, &reply)
		if err != nil {
			continue
		}
	}
	return nil
}

func (k *KawaiiIPsec) Delete() error {
	klog.Debugf("Deleting Kawaii IPsec %s", k.String())

	if k.String() == ResourceUnknown {
		return nil
	}
	// Remove parent refs
	parent, err := FindKawaiiByID(k.KawaiiID)
	if err != nil {
		klog.Errorf("Could Not find Kawaii Parent. Perhaps it was deleted ? Kawaii may have orphans")
	}
	parent.RemoveIPsec(k.String())
	return GetDB().Delete(MongoCollectionIPsecName, k.ID)
}

func (k *KawaiiIPsec) Metadata() *metadata.KawaiiIPsecConnectionMetadata {
	rules := []metadata.KawaiiFirewallRuleMetadata{}
	for _, rule := range k.Firewall.Ingress {
		metaRule := IPsecIngressRuleToMetadata(&rule)
		rules = append(rules, *metaRule)
	}
	ipsecMeta := &metadata.KawaiiIPsecConnectionMetadata{
		Name:                      k.Name,
		IP:                        k.IP,
		XfrmId:                    k.XfrmID,
		RemotePeer:                k.RemotePeer,
		RemoteSubnet:              k.RemoteSubnet,
		PreSharedKey:              k.PreSharedKey,
		DpdTimeout:                k.DpdTimeout,
		DpdTimeoutAction:          k.DpdTimeoutAction,
		StartAction:               k.StartAction,
		Rekey:                     k.Rekey,
		Phase1Lifetime:            k.Phase1Lifetime,
		Phase1DHGroup:             common.DiffieHellmanIanaNames[int(k.Phase1DHGroupNumber)],
		Phase1IntegrityAlgorithm:  k.Phase1IntegrityAlgorithm,
		Phase1EncryptionAlgorithm: k.Phase1EncryptionAlgorithm,
		Phase2Lifetime:            k.Phase2Lifetime,
		Phase2DHGroup:             common.DiffieHellmanIanaNames[int(k.Phase2DHGroupNumber)],
		Phase2IntegrityAlgorithm:  k.Phase2IntegrityAlgorithm,
		Phase2EncryptionAlgorithm: k.Phase2EncryptionAlgorithm,
		IngressRules:              rules,
	}
	return ipsecMeta
}
