/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

/* This file is a specific collection of DB schema migration or pruner helpers
   Kowabunga DB has changed over the course of time and we need to ensure that all objects
   are properly setup with the right fields before going any further.
   Check (and update when required) will only be done once, at startup.
*/

import (
	"fmt"

	"github.com/kowabunga-cloud/common/klog"
)

func kawaiiCleanupDereferencedChildren() error {
	// projects
	for _, prj := range FindProjects() {
		updated := false

		instancesToRemove := []string{}
		for _, instanceId := range prj.Instances() {
			instance, err := FindInstanceByID(instanceId)
			if err != nil || instance.String() == ResourceUnknown {
				klog.Errorf("Project %s is referencing instance %s, which doesn't seem to exists anymore, dropping it ...", prj.Name, instanceId)
				updated = true
				instancesToRemove = append(instancesToRemove, instanceId)
			}
		}

		volumesToRemove := []string{}
		for _, volumeId := range prj.Volumes() {
			volume, err := FindVolumeByID(volumeId)
			if err != nil || volume.String() == ResourceUnknown {
				klog.Errorf("Project %s is referencing volume %s, which doesn't seem to exists anymore, dropping it ...", prj.Name, volumeId)
				updated = true
				volumesToRemove = append(volumesToRemove, volumeId)
			}
		}

		komputesToRemove := []string{}
		for _, komputeId := range prj.Komputes() {
			kompute, err := FindKomputeByID(komputeId)
			if err != nil || kompute.String() == ResourceUnknown {
				klog.Errorf("Project %s is referencing Kompute %s, which doesn't seem to exists anymore, dropping it ...", prj.Name, komputeId)
				updated = true
				komputesToRemove = append(komputesToRemove, komputeId)
			}
		}

		kawaiisToRemove := []string{}
		for _, kawaiiId := range prj.Kawaiis() {
			kawaii, err := FindKawaiiByID(kawaiiId)
			if err != nil || kawaii.String() == ResourceUnknown {
				klog.Errorf("Project %s is referencing Kawaii %s, which doesn't seem to exists anymore, dropping it ...", prj.Name, kawaiiId)
				updated = true
				kawaiisToRemove = append(kawaiisToRemove, kawaiiId)
			}
		}

		kyloToRemove := []string{}
		for _, kyloId := range prj.Kylos() {
			kylo, err := FindKyloByID(kyloId)
			if err != nil || kylo.String() == ResourceUnknown {
				klog.Errorf("Project %s is referencing Kylo %s, which doesn't seem to exists anymore, dropping it ...", prj.Name, kyloId)
				updated = true
				kyloToRemove = append(kyloToRemove, kyloId)
			}
		}

		if updated {
			for _, instanceId := range instancesToRemove {
				RemoveChildRef(&prj.InstanceIDs, instanceId)
			}
			for _, volumeId := range volumesToRemove {
				RemoveChildRef(&prj.VolumeIDs, volumeId)
			}
			for _, komputeId := range komputesToRemove {
				RemoveChildRef(&prj.KomputeIDs, komputeId)
			}
			for _, kawaiiId := range kawaiisToRemove {
				RemoveChildRef(&prj.KawaiiIDs, kawaiiId)
			}
			for _, kyloId := range kyloToRemove {
				RemoveChildRef(&prj.KyloIDs, kyloId)
			}
			prj.Save()
		}
	}

	// subnets
	for _, subnet := range FindSubnets() {
		updated := false

		adaptersToRemove := []string{}
		for _, adapterId := range subnet.Adapters() {
			adapter, err := FindAdapterByID(adapterId)
			if err != nil || adapter.String() == ResourceUnknown {
				klog.Errorf("Subnet %s is referencing adapter %s, which doesn't seem to exists anymore, dropping it ...", subnet.Name, adapterId)
				updated = true
				adaptersToRemove = append(adaptersToRemove, adapterId)
			}
		}

		if updated {
			for _, adapterId := range adaptersToRemove {
				RemoveChildRef(&subnet.AdapterIDs, adapterId)
			}
			subnet.Save()
		}
	}

	return nil
}

func migrateInstances() error {
	for _, i := range FindInstances() {
		updated := false
		if i.LocalIP == "" {
			device := fmt.Sprintf("%s%d", AdapterOsNicLinuxPrefix, AdapterOsNicLinuxStartIndex+1)
			if i.OS == TemplateOsWindows {
				device = fmt.Sprintf("%s%d", AdapterOsNicWindowsPrefix, AdapterOsNicWindowsStartIndex+1)
			}

			adapterId, ok := i.Interfaces[device]
			if !ok {
				continue
			}

			adapter, err := FindAdapterByID(adapterId)
			if err != nil {
				return err
			}
			if len(adapter.Addresses) > 0 {
				i.LocalIP = adapter.Addresses[0]
				updated = true
			}
		}

		if updated {
			klog.Infof("DB Schema migration of instance %s done.", i.Name)
			i.Save()
		}
	}

	return nil
}

func dbSchemaMigration() error {
	var err error

	err = AdapterMigrateSchema()
	if err != nil {
		return err
	}

	err = AgentMigrateSchema()
	if err != nil {
		return err
	}

	err = DnsRecordMigrateSchema()
	if err != nil {
		return err
	}

	err = HarMigrateSchema()
	if err != nil {
		return err
	}

	err = InstanceMigrateSchema()
	if err != nil {
		return err
	}

	err = KaktusMigrateSchema()
	if err != nil {
		return err
	}

	err = KawaiiMigrateSchema()
	if err != nil {
		return err
	}

	err = KiwiMigrateSchema()
	if err != nil {
		return err
	}

	err = KomputeMigrateSchema()
	if err != nil {
		return err
	}

	err = KonveyMigrateSchema()
	if err != nil {
		return err
	}

	err = KyloMigrateSchema()
	if err != nil {
		return err
	}

	err = MzrMigrateSchema()
	if err != nil {
		return err
	}

	err = NfsMigrateSchema()
	if err != nil {
		return err
	}

	err = ProjectMigrateSchema()
	if err != nil {
		return err
	}

	err = RegionMigrateSchema()
	if err != nil {
		return err
	}

	err = StoragePoolMigrateSchema()
	if err != nil {
		return err
	}

	err = SubnetMigrateSchema()
	if err != nil {
		return err
	}

	err = TeamMigrateSchema()
	if err != nil {
		return err
	}

	err = TemplateMigrateSchema()
	if err != nil {
		return err
	}

	err = TokenMigrateSchema()
	if err != nil {
		return err
	}

	err = UserMigrateSchema()
	if err != nil {
		return err
	}

	err = VNetMigrateSchema()
	if err != nil {
		return err
	}

	err = VolumeMigrateSchema()
	if err != nil {
		return err
	}

	err = ZoneMigrateSchema()
	if err != nil {
		return err
	}

	return nil
}

func MigrateDatabaseSchema() error {
	klog.Infof("Migrating DB schema ...")

	err := dbSchemaMigration()
	if err != nil {
		return err
	}

	err = migrateInstances()
	if err != nil {
		return err
	}

	err = kawaiiCleanupDereferencedChildren()
	if err != nil {
		return err
	}

	return nil
}
