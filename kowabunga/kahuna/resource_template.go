/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionTemplateSchemaVersion = 2
	MongoCollectionTemplateName          = "template"

	TemplateOsWindows = "windows"
	TemplateOsLinux   = "linux"
)

type Template struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	StoragePoolID string `bson:"storage_pool_id"`

	// properties
	OS        string `bson:"os"`
	SourceURL string `bson:"source_url"`
	VolumeID  string `bson:"volume_id"`

	// children references
}

func TemplateMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("templates", MongoCollectionTemplateName)
	if err != nil {
		return err
	}

	for _, template := range FindTemplates() {
		if template.SchemaVersion == 0 || template.SchemaVersion == 1 {
			err := template.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewTemplate(poolId, name, desc, os, source string) (*Template, error) {

	switch os {
	case TemplateOsLinux, TemplateOsWindows:
		break
	default:
		os = TemplateOsLinux
	}

	t := Template{
		Resource:      NewResource(name, desc, MongoCollectionTemplateSchemaVersion),
		StoragePoolID: poolId,
		OS:            os,
		SourceURL:     source,
	}

	p, err := t.StoragePool()
	if err != nil {
		return nil, err
	}

	// ensure no such template exists in specified pool
	poolTemplates, err := p.FindTemplates()
	if err != nil {
		return nil, err
	}
	for _, tpl := range poolTemplates {
		if name == tpl.Name {
			return nil, fmt.Errorf("a template named %s already exists in pool %s", name, p.String())
		}
	}

	vol, err := NewVolume("", poolId, t.SourceURL, name, desc, VolumeTypeTemplate, 0)
	if err != nil {
		return nil, err
	}
	t.VolumeID = vol.String()

	_, err = GetDB().Insert(MongoCollectionTemplateName, t)
	if err != nil {
		// rollback, delete volume
		errD := vol.Delete()
		if errD != nil {
			klog.Error(errD)
		}

		return nil, err
	}

	klog.Debugf("Created new template %s", t.String())

	// add template to pool
	p.AddTemplate(t.String())

	return &t, nil
}

func FindTemplates() []Template {
	return FindResources[Template](MongoCollectionTemplateName)
}

func FindTemplatesByStoragePool(poolId string) ([]Template, error) {
	return FindResourcesByKey[Template](MongoCollectionTemplateName, "pool_id", poolId)
}

func FindTemplateByID(id string) (*Template, error) {
	return FindResourceByID[Template](MongoCollectionTemplateName, id)
}

func (t *Template) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionTemplateName, t.ID, from, to)
}

func (t *Template) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionTemplateName, t.ID, version)
}

func (t *Template) migrateSchemaV2() error {
	err := t.renameDbField("pool", "storage_pool_id")
	if err != nil {
		return err
	}

	err = t.renameDbField("volume", "volume_id")
	if err != nil {
		return err
	}

	err = t.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (t *Template) StoragePool() (*StoragePool, error) {
	return FindStoragePoolByID(t.StoragePoolID)
}

func (t *Template) Volume() (*Volume, error) {
	return FindVolumeByID(t.VolumeID)
}

func (t *Template) Update(name, desc string) {
	t.UpdateResourceDefaults(name, desc)
	t.Save()
}

func (t *Template) Save() {
	t.Updated()
	_, err := GetDB().Update(MongoCollectionTemplateName, t.ID, t)
	if err != nil {
		klog.Error(err)
	}
}

func (t *Template) Delete() error {
	klog.Debugf("Deleting template %s", t.String())

	if t.String() == ResourceUnknown {
		return nil
	}

	// remove template's reference from parents
	p, err := t.StoragePool()
	if err != nil {
		return err
	}

	v, err := t.Volume()
	if err != nil {
		klog.Errorf("template %s has no associated volume", t.String())
	}

	if v != nil {
		err := v.Delete()
		if err != nil {
			klog.Error(err)
			return err
		}
	}

	p.RemoveTemplate(t.String())

	return GetDB().Delete(MongoCollectionTemplateName, t.ID)
}

func (t *Template) Model() sdk.Template {
	return sdk.Template{
		Id:          t.String(),
		Name:        t.Name,
		Description: t.Description,
		Os:          t.OS,
		Source:      t.SourceURL,
	}
}
