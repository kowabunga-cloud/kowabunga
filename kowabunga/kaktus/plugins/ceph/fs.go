//go:build linux
// +build linux

/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/klog"
)

/*
   NOTE: Ceph comes with a restful API to control things but it is currently limited.
   Support for CephFS SubVolumes only has been added to release 18.x (codename 'Reef') which is not widely available.
   Consequently, we must currently rely on CLI usage for resource management, as ugly as it is
*/

type CephVolume struct {
	Name           string   `json:"name"`
	MetadataPool   string   `json:"metadata_pool"`
	MetadataPoolID int64    `json:"metadata_pool_id"`
	DataPoolIDs    []int64  `json:"data_pool_ids"`
	DataPools      []string `json:"data_pools"`
}

type CephSubVolume struct {
	Name string `json:"name"`
}

type CephSubVolumeAttr struct {
	AccessTime               string   `json:"atime"`
	BytesPercent             string   `json:"bytes_pcent"`
	BytesQuota               string   `json:"bytes_quota"`
	BytesUsed                int64    `json:"bytes_used"`
	CreatedAt                string   `json:"created_at"`
	CTime                    string   `json:"ctime"`
	DataPool                 string   `json:"data_pool"`
	Features                 []string `json:"features"`
	GID                      int64    `json:"gid"`
	Mode                     int64    `json:"mode"`
	Monitors                 []string `json:"mon_addrs"`
	ModificationTimeDataPool string   `json:"mtime"`
	Path                     string   `json:"path"`
	Namespace                string   `json:"pool_namespace"`
	State                    string   `json:"state"`
	Type                     string   `json:"type"`
	UID                      int64    `json:"uid"`
}

func (ceph *ccs) exec(args []string, js bool) (string, error) {
	if js {
		args = append(args, "-f")
		args = append(args, "json")
	}

	out, err := common.BinExecOut(ceph.Bin, "", args, []string{})
	if err != nil {
		return "", err
	}

	return out, nil
}

func (ceph *ccs) execJson(args []string, res interface{}) error {
	out, err := ceph.exec(args, true)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(out), res)
}

func (ceph *ccs) ListVolumes() ([]string, error) {
	volumes := []string{}

	cv := []CephVolume{}
	args := []string{
		"fs",
		"ls",
	}

	err := ceph.execJson(args, &cv)
	if err == nil {
		for _, v := range cv {
			volumes = append(volumes, v.Name)
		}
	}

	return volumes, err
}

func (ceph *ccs) ListSubVolumes(vol string) ([]string, error) {
	subvolumes := []string{}

	csv := []CephSubVolume{}
	args := []string{
		"fs",
		"subvolume",
		"ls",
		vol,
	}

	err := ceph.execJson(args, &csv)
	if err == nil {
		for _, s := range csv {
			subvolumes = append(subvolumes, s.Name)
		}
	}

	return subvolumes, err
}

func (ceph *ccs) subVolume(vol, sub string) (CephSubVolumeAttr, error) {

	sv := CephSubVolumeAttr{}
	args := []string{
		"fs",
		"subvolume",
		"info",
		vol,
		sub,
	}

	err := ceph.execJson(args, &sv)
	return sv, err
}

func (ceph *ccs) CreateSubVolume(vol, sub string) (string, int64, error) {

	args := []string{
		"fs",
		"subvolume",
		"create",
		vol,
		sub,
	}

	_, err := ceph.exec(args, false)
	if err != nil {
		klog.Error(err)
		return "", 0, err
	}

	sv, err := ceph.subVolume(vol, sub)
	if err != nil {
		return "", 0, err
	}

	return sv.Path, sv.BytesUsed, nil
}

func (ceph *ccs) DeleteSubVolume(vol, sub string) error {

	args := []string{
		"fs",
		"subvolume",
		"rm",
		vol,
		sub,
	}

	_, err := ceph.exec(args, false)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}
