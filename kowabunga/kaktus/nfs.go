/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"

	"github.com/kowabunga-cloud/common/klog"
)

type NfsExport struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	FS        string   `json:"fs"`
	Path      string   `json:"path"`
	Access    string   `json:"access"`
	Protocols []int32  `json:"protocols"`
	Clients   []string `json:"clients"`
}

func NewNfsExport(idStr, name, fs, path, access string, protocols []int32, clients []string) NfsExport {

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return NfsExport{}
	}

	return NfsExport{
		ID:        id,
		Name:      "/" + name,
		FS:        fs,
		Path:      path,
		Access:    access,
		Protocols: protocols,
		Clients:   clients,
	}
}

func (n *NfsExport) CreateBackend(ip string, port int) error {
	uri := fmt.Sprintf("http://%s:%d/api/v1/export", ip, port)

	client := resty.New()
	_, err := client.R().SetBody(n).Post(uri)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (n *NfsExport) UpdateBackend(ip string, port int) error {
	uri := fmt.Sprintf("http://%s:%d/api/v1/export/%d", ip, port, n.ID)

	client := resty.New()
	_, err := client.R().SetBody(n).Put(uri)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (n *NfsExport) DeleteBackend(ip string, port int) error {
	uri := fmt.Sprintf("http://%s:%d/api/v1/export/%d", ip, port, n.ID)

	client := resty.New()
	_, err := client.R().Delete(uri)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

type NfsConnectionSettings struct{}

func NewNfsConnectionSettings() (*NfsConnectionSettings, error) {
	return &NfsConnectionSettings{}, nil
}

func (ncs *NfsConnectionSettings) CreateBackends(idStr, name, fs, path, access string, protocols []int32, clients, backends []string, port int) error {
	export := NewNfsExport(idStr, name, fs, path, access, protocols, clients)
	for _, b := range backends {
		klog.Debugf("Creating NFS export %s on %s:%d", name, b, port)
		err := export.CreateBackend(b, port)
		if err != nil {
			klog.Error(err)
		}
	}
	return nil
}

func (ncs *NfsConnectionSettings) UpdateBackends(idStr, name, fs, path, access string, protocols []int32, clients, backends []string, port int) error {
	export := NewNfsExport(idStr, name, fs, path, access, protocols, clients)
	for _, b := range backends {
		klog.Debugf("Updating export on %s %s:%d", name, b, port)
		err := export.UpdateBackend(b, port)
		if err != nil {
			klog.Error(err)
		}
	}
	return nil
}

func (ncs *NfsConnectionSettings) DeleteBackends(idStr, name, fs, path, access string, protocols []int32, clients, backends []string, port int) error {
	export := NewNfsExport(idStr, name, fs, path, access, protocols, clients)
	for _, b := range backends {
		klog.Debugf("Deleting export on %s %s:%d", name, b, port)
		err := export.DeleteBackend(b, port)
		if err != nil {
			klog.Error(err)
		}
	}
	return nil
}
