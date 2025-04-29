/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

type KowabungaDB struct {
	Client *mongo.Client
	DB     *mongo.Database
}

type KowabungaDbEvent struct {
	DocumentKey KowabungaDocumentKey `bson:"documentKey"`
	Operation   string               `bson:"operationType"`
}
type KowabungaDocumentKey struct {
	ID primitive.ObjectID `bson:"_id"`
}

// database singleton
var dbLock = &sync.Mutex{}
var kDB *KowabungaDB

func GetDB() *KowabungaDB {
	if kDB == nil {
		dbLock.Lock()
		defer dbLock.Unlock()
		klog.Debugf("Creating Kowabunga DB instance")
		kDB = &KowabungaDB{}
	}

	return kDB
}

func (db *KowabungaDB) Open(uri, database string) error {

	client, err := mongo.NewClient(options.Client().ApplyURI(uri).SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
	if err != nil {
		return err
	}
	db.Client = client

	// connecting to DB
	err = db.Client.Connect(context.TODO())
	if err != nil {
		return err
	}

	// look for a primary server
	err = db.Client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return err
	}

	db.DB = db.Client.Database(database)

	return nil
}

func (db *KowabungaDB) Admin() *mongo.Database {
	return db.Client.Database("admin")
}

func (db *KowabungaDB) Close() error {
	if db.Client != nil {
		return db.Client.Disconnect(context.TODO())
	}
	return nil
}

func (db *KowabungaDB) HasCollection(collection string) bool {
	coll, _ := db.DB.ListCollectionNames(context.Background(), bson.D{primitive.E{Key: "name", Value: collection}})
	return len(coll) == 1
}

func (db *KowabungaDB) RenameCollection(from, to string) error {
	dbAdmin := db.Admin()
	if !db.HasCollection(from) {
		return nil
	}
	_from := fmt.Sprintf("%s.%s", db.DB.Name(), from)
	_to := fmt.Sprintf("%s.%s", db.DB.Name(), to)
	klog.Infof("Renaming MongoDB collection '%s' into '%s'", _from, _to)
	return dbAdmin.RunCommand(context.Background(), bson.D{primitive.E{Key: "renameCollection", Value: _from}, primitive.E{Key: "to", Value: _to}}).Err()
}

func (db *KowabungaDB) Insert(collection string, obj interface{}) (interface{}, error) {
	c := db.DB.Collection(collection)
	return c.InsertOne(context.TODO(), obj)
}

func (db *KowabungaDB) Update(collection string, id primitive.ObjectID, obj interface{}) (interface{}, error) {
	// cleanup cache data, if any
	defer func() {
		_ = GetCache().Delete(collection, id.Hex())
	}()

	c := db.DB.Collection(collection)
	return c.ReplaceOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: id}}, obj)
}

func (db *KowabungaDB) Rename(collection string, id primitive.ObjectID, from, to string) error {
	// cleanup cache data, if any
	defer func() {
		_ = GetCache().Delete(collection, id.Hex())
	}()

	c := db.DB.Collection(collection)

	// check if document has such a field
	result := c.FindOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: id}, primitive.E{Key: from, Value: bson.D{primitive.E{Key: "$exists", Value: true}}}})
	if result.Err() == nil {
		// it does: rename field and update document
		klog.Debugf("Renaming document '%s' from '%s' field '%s' into '%s'", id.Hex(), collection, from, to)
		filter := bson.D{primitive.E{Key: "_id", Value: id}}
		update := bson.D{primitive.E{Key: "$rename", Value: bson.D{primitive.E{Key: from, Value: to}}}}
		_, err := c.UpdateOne(context.TODO(), filter, update)
		return err
	}

	return nil
}

func (db *KowabungaDB) SetSchemaVersion(collection string, id primitive.ObjectID, schemaVersion int) error {
	// cleanup cache data, if any
	defer func() {
		_ = GetCache().Delete(collection, id.Hex())
	}()

	c := db.DB.Collection(collection)
	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.D{primitive.E{Key: "$set", Value: bson.D{primitive.E{Key: "schema_version", Value: schemaVersion}}}}

	// check if document has such a schemaVersion
	result1 := c.FindOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: id}, primitive.E{Key: "schema_version", Value: bson.D{primitive.E{Key: "$exists", Value: false}}}})
	if result1.Err() == nil {
		// it does not: adds initial schema version
		klog.Debugf("Updating document '%s' from '%s', initializing 'schemaVersion' field to '%d'", id.Hex(), collection, schemaVersion)
		_, err := c.UpdateOne(context.TODO(), filter, update)
		return err
	}

	// check if document has outdated schemaVersion
	result2 := c.FindOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: id}, primitive.E{Key: "schema_version", Value: bson.D{primitive.E{Key: "$ne", Value: schemaVersion}}}})
	if result2.Err() == nil {
		// it does: upates schema version
		klog.Debugf("Updating document '%s' from '%s', setting 'schemaVersion' field to '%d'", id.Hex(), collection, schemaVersion)
		_, err := c.UpdateOne(context.TODO(), filter, update)
		return err
	}

	return nil
}

func (db *KowabungaDB) FindAll(collection string, results interface{}) error {
	c := db.DB.Collection(collection)
	cursor, err := c.Find(context.TODO(), bson.D{})
	if err != nil {
		return err
	}

	return cursor.All(context.TODO(), results)
}

func (db *KowabungaDB) FindAllByKey(collection, key, value string, results interface{}) error {
	c := db.DB.Collection(collection)
	cursor, err := c.Find(context.TODO(), bson.D{primitive.E{Key: key, Value: value}})
	if err != nil {
		return err
	}

	return cursor.All(context.TODO(), results)
}

func (db *KowabungaDB) Find(collection, k, v string, result interface{}) error {
	c := db.DB.Collection(collection)
	return c.FindOne(context.TODO(), bson.D{primitive.E{Key: k, Value: v}}, nil).Decode(result)
}

func (db *KowabungaDB) FindByArrayContains(collection, k, v string, result interface{}) error {
	c := db.DB.Collection(collection)
	return c.FindOne(context.TODO(), bson.D{primitive.E{Key: k, Value: "{$all: [" + v + "]}"}}, nil).Decode(result)
}

func (db *KowabungaDB) FindByID(collection, id string, result interface{}) error {
	// look into cache first
	err := GetCache().Get(collection, id, &result)
	if err == nil {
		return nil
	}

	// failover: look into DB
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	c := db.DB.Collection(collection)
	err = c.FindOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: oid}}, nil).Decode(result)
	if err == nil {
		// cache back data
		GetCache().Set(collection, id, result)
	}

	return nil
}

func (db *KowabungaDB) FindByName(collection, name string, result interface{}) error {
	return db.Find(collection, "name", name, result)
}

func (db *KowabungaDB) FindByType(collection, tp string, result interface{}) error {
	return db.Find(collection, "type", tp, result)
}

func (db *KowabungaDB) FindByEmail(collection, email string, result interface{}) error {
	return db.Find(collection, "email", email, result)
}

func (db *KowabungaDB) FindByIP(collection, ip string, result interface{}) error {
	return db.Find(collection, "local_ip", ip, result)
}

func (db *KowabungaDB) Delete(collection string, id primitive.ObjectID) error {
	c := db.DB.Collection(collection)
	_, err := c.DeleteOne(context.TODO(), bson.D{primitive.E{Key: "_id", Value: id}})
	return err
}
