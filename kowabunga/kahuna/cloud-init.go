/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kdomanski/iso9660"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/klog"
)

const (
	CloudInitUserData       = "user-data"
	CloudInitMetaData       = "meta-data"
	CloudInitNetworkConfig  = "network-config"
	CloudinitProfileLinux   = "linux"
	CloudinitProfileKawaii  = "kawaii"
	CloudinitProfileKonvey  = "konvey"
	CloudinitProfileWindows = "windows"
	CloudInitVolumeSuffix   = "-cloudinit"
)

type CloudInit struct {
	Name          string
	OS            string
	TmpDir        string
	IsoImage      string
	IsoSize       int64
	UserData      string
	MetaData      string
	NetworkConfig string
}

func NewCloudInit(name, osType string) (*CloudInit, error) {
	dir, err := os.MkdirTemp("", "cloud-init")
	if err != nil {
		return nil, err
	}

	// generate random suffix
	volName := name + CloudInitVolumeSuffix
	return &CloudInit{
		Name:   volName,
		OS:     osType,
		TmpDir: dir,
	}, nil
}

func (ci *CloudInit) SetData(src, dst string, values any) error {

	tpl := template.New(filepath.Base(src))
	common.LoadTemplateFunctions(tpl)

	tpl, err := tpl.ParseFiles(src)
	if err != nil {
		return err
	}

	out, err := os.Create(filepath.Clean(filepath.Join(ci.TmpDir, dst)))
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	// Render Template with input values
	err = tpl.Execute(out, values)
	if err != nil {
		klog.Error(err)
	}

	return nil
}

type UserDataSettings struct {
	Hostname          string
	Domain            string
	RootPassword      string
	ServiceUser       string
	ServiceUserPubKey string
	Profile           string
	MetadataAlias     string
	InterfacesSubnet  map[string]Subnet
}

func (ci *CloudInit) SetUserData(name, domain, pwd, user, pubkey, profile string, adapters []string) error {
	metadataAlias := fmt.Sprintf(`#!/bin/bash

KW_ENDPOINT="$(cloud-init query ds.meta_data.kowabunga_metadata_uri)"
KW_INSTANCE_ID="$(cloud-init query instance_id)"
KW_LOCAL_IP="$(cloud-init query ds.meta_data.kowabunga_local_ip)"

curl -s ${KW_ENDPOINT} -H "%s: ${KW_LOCAL_IP}" -H "%s: ${KW_INSTANCE_ID}"
`, common.HttpHeaderKowabungaSourceIP, common.HttpHeaderKowabungaInstanceID)

	data := UserDataSettings{
		Hostname:          name,
		Domain:            domain,
		RootPassword:      pwd,
		ServiceUser:       user,
		ServiceUserPubKey: pubkey,
		Profile:           profile,
		MetadataAlias:     metadataAlias,
	}
	tpl := GetCfg().CloudInit.Linux.UserData
	routesByInterface := make(map[string]Subnet)
	if ci.OS == TemplateOsWindows {
		// On windows, cloudbase init does not support routes except from gateways

		for idx, adapterId := range adapters {
			a, err := FindAdapterByID(adapterId)
			if err != nil {
				return err
			}

			s, err := a.Subnet()
			if err != nil {
				return err
			}
			deviceName := ci.formatAdapterDeviceName(idx, true)
			routesByInterface[deviceName] = *s
		}
		tpl = GetCfg().CloudInit.Windows.UserData
	}
	data.InterfacesSubnet = routesByInterface
	return ci.SetData(tpl, CloudInitUserData, data)
}

type MetaDataSettings struct {
	Profile              string
	Region               string
	Zone                 string
	InstanceID           string
	Hostname             string
	MetadataEndpoint     string
	ControllerEndpoint   string
	LocalIP              string
	ControllerAgentID    string
	ControllerAgentToken string
}

func (ci *CloudInit) SetMetaData(profile, zoneId, instanceId, agentId, instanceName, localIP string) error {
	zone, err := FindZoneByID(zoneId)
	if err != nil {
		return err
	}

	region, err := zone.Region()
	if err != nil {
		return err
	}

	metadataEndpoint := fmt.Sprintf("%s/latest/meta-data", GetCfg().Global.PublicURL)
	data := MetaDataSettings{
		Profile:          profile,
		Region:           region.Name,
		Zone:             zone.Name,
		InstanceID:       instanceId,
		Hostname:         instanceName,
		MetadataEndpoint: metadataEndpoint,
		LocalIP:          localIP,
	}

	if agentId != "" {
		a, err := FindAgentByID(agentId)
		if err != nil {
			return err
		}

		// check if instance already has a registered token
		var t *Token

		tokenName := fmt.Sprintf("%s-api-key", a.Name)
		t, err = FindTokenByName(tokenName)
		if err != nil {
			// can't find any token, will create a new one
			t, err = NewAgentToken(agentId, tokenName, "", false, "")
			if err != nil {
				return err
			}
		}
		token, err := t.SetNewApiKey(false)
		if err != nil {
			return err
		}

		// disconnect any live agent WebSocket, if any
		DisconnectAgent(agentId)

		data.ControllerEndpoint = strings.Replace(GetCfg().Global.PublicURL, "http", "ws", 1)
		data.ControllerAgentID = agentId
		data.ControllerAgentToken = token
	}

	tpl := GetCfg().CloudInit.Linux.MetaData
	if ci.OS == TemplateOsWindows {
		tpl = GetCfg().CloudInit.Windows.MetaData
	}
	return ci.SetData(tpl, CloudInitMetaData, data)
}

type NetworkConfigSettings struct {
	Device          string
	MAC             string
	Private         bool
	Addresses       []string
	GatewayEnabled  bool
	VLANGateway     string
	InternetGateway string
	Profile         string
	DNS             string
	Domain          string
	Routes          []string
}

func (ci *CloudInit) SetNetworkConfig(projectId, zoneId, domain, profile string, adapters []string) error {

	p, err := FindProjectByID(projectId)
	if err != nil {
		return err
	}

	zoneGw, err := p.GetZoneGatewayAddress(zoneId)
	if err != nil {
		return err
	}

	data := []NetworkConfigSettings{}
	count := 0
	for idx, adapterId := range adapters {
		a, err := FindAdapterByID(adapterId)
		if err != nil {
			return err
		}

		s, err := a.Subnet()
		if err != nil {
			return err
		}

		ip, ipnet, err := net.ParseCIDR(s.CIDR)
		if err != nil {
			return err
		}

		// Set local-zone Kawaii virtual IP if adapter's subnet is project's main one
		// For peering adapters, no gateway is required
		gwEnabled := false
		gw_ip := ""
		gwIP := net.ParseIP(zoneGw)
		if gwIP == nil {
			return fmt.Errorf("invalid IP: %s", gwIP)
		}
		if ipnet.Contains(gwIP) {
			gw_ip = zoneGw
			gwEnabled = true
		}

		dom := ""
		private := ip.IsPrivate()
		if private {
			dom = domain
		} else {
			gwEnabled = true
		}

		dev := ci.formatAdapterDeviceName(idx, false)
		if ci.OS == TemplateOsWindows {
			dev = ci.formatAdapterDeviceName(idx, true)
		}
		cfg := NetworkConfigSettings{
			Device:          dev,
			MAC:             a.MAC,
			Private:         private,
			Profile:         profile,
			Addresses:       []string{},
			GatewayEnabled:  gwEnabled,
			InternetGateway: gw_ip,
			VLANGateway:     s.Gateway,
			DNS:             s.DNS,
			Domain:          dom,
			Routes:          s.Routes,
		}

		if profile == CloudinitProfileKawaii {
			// instances have 2 main interfaces: WAN (first) and LAN (second)
			// optional private ones are VPC peering and should not be associated
			// with routes or DNS config
			if count >= 2 {
				cfg.GatewayEnabled = false
				cfg.Routes = []string{}
				cfg.Domain = ""
				cfg.DNS = ""
			}
		}

		mask, _ := ipnet.Mask.Size()
		for _, addr := range a.Addresses {
			ip := fmt.Sprintf("%s/%d", addr, mask)
			cfg.Addresses = append(cfg.Addresses, ip)
		}

		data = append(data, cfg)
		count += 1
	}

	tpl := GetCfg().CloudInit.Linux.NetworkConfig
	if ci.OS == TemplateOsWindows {
		tpl = GetCfg().CloudInit.Windows.NetworkConfig
	}
	return ci.SetData(tpl, CloudInitNetworkConfig, data)
}

func (ci *CloudInit) WriteISO() error {
	wr, err := iso9660.NewWriter()
	if err != nil {
		return err
	}
	defer func() {
		_ = wr.Cleanup()
	}()

	err = wr.AddLocalDirectory(ci.TmpDir, "/")
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "cloud-init")
	if err != nil {
		return err
	}
	ci.IsoImage = f.Name()

	klog.Debugf("Saving cloud-init ISO image into %s", ci.IsoImage)
	err = wr.WriteTo(f, "cidata")
	if err != nil {
		return err
	}

	infos, err := f.Stat()
	if err != nil {
		return err
	}
	ci.IsoSize = infos.Size()

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func (ci *CloudInit) Delete() error {
	// remove leftover files
	err := os.RemoveAll(ci.TmpDir)
	if err != nil {
		return err
	}

	return os.Remove(ci.IsoImage)
}

func (ci *CloudInit) formatAdapterDeviceName(index int, isWindows bool) string {
	if isWindows {
		return fmt.Sprintf("%s%d", AdapterOsNicWindowsPrefix, index+AdapterOsNicWindowsStartIndex)
	}
	return fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, index+AdapterOsNicLinuxStartIndex)
}
