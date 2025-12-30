/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"time"

	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionTokenSchemaVersion = 2
	MongoCollectionTokenName          = "token"

	TokenParentTypeAgent = "agent"
	TokenParentTypeUser  = "user"

	TokenApiKeyLength                = 64
	TokenApiKeyDigitsCount           = 16
	TokenApiKeySymbolsCount          = 0
	TokenApiKeyLowercaseOnly         = false
	TokenApiKeyAllowRepeatCharacters = true
	TokenApiKeyHashCost              = 10
)

type Token struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	AgentID string `bson:"agent_id"`
	UserID  string `bson:"user_id"`

	// properties
	ParentType     string `bson:"parent_type"`
	Expire         bool   `bson:"expire"`
	ExpirationDate string `bson:"expiration_date"`
	ApiKeyHash     string `bson:"api_key_hash"`

	// children references
}

func TokenMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("tokens", MongoCollectionTokenName)
	if err != nil {
		return err
	}

	for _, token := range FindTokens() {
		if token.SchemaVersion == 0 || token.SchemaVersion == 1 {
			err := token.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func newToken(name, desc string, expire bool, expirationDate string) (*Token, error) {
	t := Token{
		Resource: NewResource(name, desc, MongoCollectionTokenSchemaVersion),
	}

	err := t.SetExpirationDate(expire, expirationDate)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func NewAgentToken(agentId, name, desc string, expire bool, expirationDate string) (*Token, error) {
	t, err := newToken(name, desc, expire, expirationDate)
	if err != nil {
		return nil, err
	}

	t.ParentType = TokenParentTypeAgent
	t.AgentID = agentId

	a, err := t.Agent()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionTokenName, t)
	if err != nil {
		return nil, err
	}
	klog.Debugf("Created new agent token %s", t.String())

	// add token to agent
	a.AddToken(t.String())

	return t, nil
}

func NewUserToken(userId, name, desc string, expire bool, expirationDate string) (*Token, error) {
	t, err := newToken(name, desc, expire, expirationDate)
	if err != nil {
		return nil, err
	}

	t.ParentType = TokenParentTypeUser
	t.UserID = userId

	u, err := t.User()
	if err != nil {
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionTokenName, t)
	if err != nil {
		return nil, err
	}
	klog.Debugf("Created new user token %s", t.String())

	// add token to agent
	u.AddToken(t.String())

	return t, nil
}

func FindTokens() []Token {
	return FindResources[Token](MongoCollectionTokenName)
}

func FindTokenByID(id string) (*Token, error) {
	return FindResourceByID[Token](MongoCollectionTokenName, id)
}

func FindTokenByName(name string) (*Token, error) {
	return FindResourceByName[Token](MongoCollectionTokenName, name)
}

func FindTokensByAgent(agentId string) ([]Token, error) {
	return FindResourcesByKey[Token](MongoCollectionTokenName, "agent_id", agentId)
}

func (t *Token) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionTokenName, t.ID, from, to)
}

func (t *Token) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionTokenName, t.ID, version)
}

func (t *Token) migrateSchemaV2() error {
	err := t.renameDbField("agent", "agent_id")
	if err != nil {
		return err
	}

	err = t.renameDbField("user", "user_id")
	if err != nil {
		return err
	}

	err = t.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (t *Token) Agent() (*Agent, error) {
	return FindAgentByID(t.AgentID)
}

func (t *Token) User() (*User, error) {
	return FindUserByID(t.UserID)
}

func (t *Token) SetExpirationDate(expire bool, expirationDate string) error {
	if expire {
		// check for valid expiration date format
		_, err := time.Parse(time.DateOnly, expirationDate)
		if err != nil {
			return err
		}
		t.Expire = true
		t.ExpirationDate = expirationDate
	}

	return nil
}

func (t *Token) HasExpired() bool {
	if t.Expire {
		ed, err := time.Parse(time.DateOnly, t.ExpirationDate)
		if err != nil {
			return true
		}

		if ed.Unix() < time.Now().Unix() {
			return true
		}
	}

	return false
}

func (t *Token) SetNewApiKey(notify bool) (string, error) {
	// generate a new robust api key
	apiKey, err := password.Generate(TokenApiKeyLength, TokenApiKeyDigitsCount,
		TokenApiKeySymbolsCount, TokenApiKeyLowercaseOnly, TokenApiKeyAllowRepeatCharacters)
	if err != nil {
		return "", fmt.Errorf("unable to generate new robust api key: %v", err)
	}

	// generate "hash" for DB storage
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), TokenApiKeyHashCost)
	if err != nil {
		return apiKey, fmt.Errorf("unable to generate hash from generated api key: %v", err)
	}

	t.ApiKeyHash = string(hash)
	t.Save()

	if notify {
		// send plain-text API key by email, will only happen once, there's no way to recover from it
		switch t.ParentType {
		case TokenParentTypeAgent:
			agent, err := t.Agent()
			if err == nil {
				_ = NewEmailAgentApiToken(agent, apiKey)
			}
		case TokenParentTypeUser:
			user, err := t.User()
			if err == nil {
				_ = NewEmailUserApiToken(user, apiKey)
			}
		}
	}

	return apiKey, nil
}

func (t *Token) Verify(apiKey string) error {
	// comparing the supplied api key with the hashed one from database
	return bcrypt.CompareHashAndPassword([]byte(t.ApiKeyHash), []byte(apiKey))
}

func (t *Token) Update(name, desc string, expire bool, expirationDate string) error {
	t.UpdateResourceDefaults(name, desc)

	err := t.SetExpirationDate(expire, expirationDate)
	if err != nil {
		return err
	}

	t.Save()
	return nil
}

func (t *Token) Save() {
	t.Updated()
	_, err := GetDB().Update(MongoCollectionTokenName, t.ID, t)
	if err != nil {
		klog.Error(err)
	}
}

func (t *Token) Delete() error {
	klog.Debugf("Deleting token %s", t.String())

	if t.String() == ResourceUnknown {
		return nil
	}

	// remove token's reference from parents, one of those
	switch t.ParentType {
	case TokenParentTypeAgent:
		a, err := t.Agent()
		if err != nil {
			return err
		}
		a.RemoveToken(t.String())
	default:
	}

	return GetDB().Delete(MongoCollectionTokenName, t.ID)
}

func (t *Token) Model() sdk.ApiToken {
	return sdk.ApiToken{
		Id:             t.String(),
		Name:           t.Name,
		Description:    t.Description,
		Expire:         t.Expire,
		ExpirationDate: t.ExpirationDate,
	}
}
