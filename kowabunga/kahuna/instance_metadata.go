/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/metadata"
)

func GetInstanceMetadata(srcIp, instanceId string) (metadata.InstanceMetadata, error) {
	meta := metadata.InstanceMetadata{}
	var err error
	i, err := FindInstanceByIP(srcIp)
	if err != nil {
		return meta, err
	}

	if i.String() != instanceId {
		return meta, fmt.Errorf("mismatch between instance ID and local IP at metadata retrieval")
	}

	meta.ID = i.String()
	meta.Name = i.Name
	meta.Description = i.Description
	meta.VCPUs = i.CPU
	meta.Memory = bytesToGB(i.Memory)
	meta.LocalIPv4 = i.LocalIP
	meta.Cost = fmt.Sprintf("%f %s", i.Cost.Price, i.Cost.Currency)

	for _, volumeId := range i.Volumes() {
		v, err := FindVolumeByID(volumeId)
		if err != nil {
			return meta, err
		}

		if v.Type == VolumeTypeOs {
			template, err := v.Template()
			if err != nil {
				return meta, err
			}
			meta.Template = template.Name
		}
	}

	p, err := i.Project()
	if err != nil {
		return meta, err
	}

	meta.Project = p.Name
	meta.Domain = p.Domain
	meta.Tags = p.Tags

	k, err := i.Kaktus()
	if err != nil {
		return meta, err
	}
	meta.Kaktus = k.Name

	z, err := k.Zone()
	if err != nil {
		return meta, err
	}
	meta.Zone = z.Name

	r, err := z.Region()
	if err != nil {
		return meta, err
	}

	meta.Region = r.Name

	// find private subnet
	privateSubnetId, err := p.GetPrivateSubnet(r.String())
	if err != nil {
		return meta, err
	}
	subnet, err := FindSubnetByID(privateSubnetId)
	if err != nil {
		return meta, err
	}
	meta.SubnetCIDR = subnet.CIDR

	switch i.Profile {
	case "kgw", CloudinitProfileKawaii:
		kawaii, err := FindKawaiiByID(i.ProfileID)
		if err == nil {
			m := kawaii.Metadata(i.String())
			meta.Kawaii = &m
			meta.Peers, err = getMZRPeers(kawaii, i)
			if err != nil {
				klog.Error(err)
			}
		}
	case CloudinitProfileKonvey:
		konvey, err := FindKonveyByID(i.ProfileID)
		if err == nil {
			m := konvey.Metadata(i.String())
			meta.Konvey = &m
		}
	}

	return meta, err
}

func getMZRPeers(mzrObject MZR, instance *Instance) ([]string, error) {
	var peers []string
	mzr, err := mzrObject.MZR()
	if err != nil {
		return peers, fmt.Errorf("Metadata MZR %d : %w", mzr.ID, err)
	}
	ips, err := mzr.FindLocalPrivateIPs()
	if err != nil {
		return peers, fmt.Errorf("Metadata MZR %d : %w", mzr.ID, err)
	}
	for _, ip := range ips {
		if ip != instance.LocalIP {
			peers = append(peers, ip)
		}
	}
	return peers, nil
}
