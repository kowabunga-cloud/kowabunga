/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionAgentSchemaVersion = 2
	MongoCollectionAgentName          = "agent"

	ErrAgentNoSuchToken = "no such token in agent"
)

type Agent struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents

	// properties
	Type    string `bson:"type"`
	TokenID string `bson:"token_id"`

	// children references
}

func AgentMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("agents", MongoCollectionAgentName)
	if err != nil {
		return err
	}

	for _, agent := range FindAgents() {
		if agent.SchemaVersion == 0 || agent.SchemaVersion == 1 {
			err := agent.migrateSchemaV2()
			if err != nil {
				return err
			}

			// migrate data
			agentReloaded, err := FindAgentByID(agent.String())
			if err != nil {
				return err
			}

			switch agentReloaded.Type {
			case "KSA":
				// delete: KSA/KCA are now merged in Kaktus
				err := agentReloaded.Delete()
				if err != nil {
					return err
				}
				continue
			case "KCA":
				agentReloaded.Type = common.KowabungaKaktusAgent
				agentReloaded.Save()
			case "KNA":
				agentReloaded.Type = common.KowabungaKiwiAgent
				agentReloaded.Save()
			}

		}
	}

	return nil
}

func NewAgent(name, desc, kind string) (*Agent, error) {
	a := Agent{
		Resource: NewResource(name, desc, MongoCollectionAgentSchemaVersion),
		Type:     kind,
		TokenID:  "",
	}

	_, err := GetDB().Insert(MongoCollectionAgentName, a)
	if err != nil {
		return nil, err
	}
	klog.Debugf("Created new agent %s %s", kind, a.String())

	return &a, nil
}

func FindAgents() []Agent {
	return FindResources[Agent](MongoCollectionAgentName)
}

func FindAgentByID(id string) (*Agent, error) {
	return FindResourceByID[Agent](MongoCollectionAgentName, id)
}

func FindAgentByName(name string) (*Agent, error) {
	return FindResourceByName[Agent](MongoCollectionAgentName, name)
}

func (a *Agent) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionAgentName, a.ID, from, to)
}

func (a *Agent) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionAgentName, a.ID, version)
}

func (a *Agent) migrateSchemaV2() error {
	err := a.renameDbField("token", "token_id")
	if err != nil {
		return err
	}

	err = a.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) Token() (*Token, error) {
	return FindTokenByID(a.TokenID)
}

func (a *Agent) Update(name, desc string) {
	a.UpdateResourceDefaults(name, desc)
	a.Save()
}

func (a *Agent) Save() {
	a.Updated()
	_, err := GetDB().Update(MongoCollectionAgentName, a.ID, a)
	if err != nil {
		klog.Error(err)
	}
}

func (a *Agent) Delete() error {
	klog.Debugf("Deleting agent %s", a.String())

	if a.String() == ResourceUnknown {
		return nil
	}

	t, err := a.Token()
	if err != nil {
		return err
	}

	err = t.Delete()
	if err != nil {
		return err
	}

	return GetDB().Delete(MongoCollectionAgentName, a.ID)
}

func (a *Agent) Model() sdk.Agent {
	return sdk.Agent{
		Id:          a.String(),
		Name:        a.Name,
		Description: a.Description,
		Type:        a.Type,
	}
}

// Tokens

func (a *Agent) AddToken(id string) {
	klog.Debugf("Adding token %s to agent %s", id, a.String())
	a.TokenID = id
	a.Save()
}

func (a *Agent) RemoveToken(id string) {
	klog.Debugf("Removing token %s from agent %s", id, a.String())
	a.TokenID = ""
	a.Save()
}
