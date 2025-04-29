/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	CostCurrencyDefault = "EUR"
	CostErrorUnknown    = "Unknown Cost"

	ResourceUnknown = "000000000000000000000000"
)

type Resources struct {
	IDs []string
}

type Resource struct {
	ID            primitive.ObjectID `bson:"_id"`
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
	SchemaVersion int                `bson:"schema_version"`

	Name        string `bson:"name"`
	Description string `bson:"description"`
}

func NewResource(name, desc string, schemaVersion int) Resource {
	now := time.Now()
	return Resource{
		ID:            primitive.NewObjectIDFromTimestamp(now),
		CreatedAt:     now,
		UpdatedAt:     now,
		SchemaVersion: schemaVersion,
		Name:          name,
		Description:   desc,
	}
}

func (r *Resource) String() string {
	return r.ID.Hex()
}

func (r *Resource) Updated() {
	r.UpdatedAt = time.Now()
}

func (r *Resource) UpdateResourceDefaults(name, desc string) {
	SetFieldStr(&r.Name, name)
	SetFieldStr(&r.Description, desc)
}

func FindResources[T any](collection string) []T {
	var res []T
	err := GetDB().FindAll(collection, &res)
	if err != nil {
		klog.Error(err)
	}
	return res
}

func FindResourceByID[T any](collection, id string) (*T, error) {
	var res T
	err := GetDB().FindByID(collection, id, &res)
	return &res, err
}

func FindResourceByName[T any](collection, name string) (*T, error) {
	var res T
	err := GetDB().FindByName(collection, name, &res)
	return &res, err
}

func FindResourceByType[T any](collection, tp string) (*T, error) {
	var res T
	err := GetDB().FindByType(collection, tp, &res)
	return &res, err
}

func FindResourceByEmail[T any](collection, email string) (*T, error) {
	var res T
	err := GetDB().FindByEmail(collection, email, &res)
	return &res, err
}

func FindResourceByIP[T any](collection, ip string) (*T, error) {
	var res T
	err := GetDB().FindByIP(collection, ip, &res)
	return &res, err
}

func FindResourcesByKey[T any](collection, key, value string) ([]T, error) {
	var res []T
	err := GetDB().FindAllByKey(collection, key, value, &res)
	return res, err
}

// Cost

type CostStructure struct {
	Capacity int64
	Cost     ResourceCost
}

type ResourceCost struct {
	Price    float32 `bson:"cost,truncate"`
	Currency string  `bson:"currency"`
}

func NewResourceCost(price float32, currency string) ResourceCost {
	c := CostCurrencyDefault
	if currency != "" {
		c = currency
	}
	return ResourceCost{
		Price:    price,
		Currency: c,
	}
}

func (r *ResourceCost) Model() sdk.Cost {
	return sdk.Cost{
		Price:    r.Price,
		Currency: r.Currency,
	}
}

// Metadata

type ResourceMetadata struct {
	Key   string `bson:"key"`
	Value string `bson:"value"`
}

func NewResourceMetadata(k, v string) ResourceMetadata {
	return ResourceMetadata{
		Key:   k,
		Value: v,
	}
}

func (r *ResourceMetadata) Model() sdk.Metadata {
	return sdk.Metadata{
		Key:   r.Key,
		Value: r.Value,
	}
}
