/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"slices"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionTeamSchemaVersion = 2
	MongoCollectionTeamName          = "team"
	//ErrUserNoSuchToken   = "no such token in user"
)

type Team struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents

	// properties

	// children references
	UserIDs []string `bson:"user_ids"`
}

func TeamMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("groups", MongoCollectionTeamName)
	if err != nil {
		return err
	}

	for _, team := range FindTeams() {
		if team.SchemaVersion == 0 || team.SchemaVersion == 1 {
			err := team.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewTeam(name, desc string, users []string) (*Team, error) {
	g := Team{
		Resource: NewResource(name, desc, MongoCollectionTeamSchemaVersion),
		UserIDs:  []string{},
	}

	// verify users
	g.SetUsers(users)

	_, err := GetDB().Insert(MongoCollectionTeamName, g)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new users team %s (%s)", g.String(), g.Name)

	return &g, nil
}

func FindTeams() []Team {
	return FindResources[Team](MongoCollectionTeamName)
}

func FindTeamByID(id string) (*Team, error) {
	return FindResourceByID[Team](MongoCollectionTeamName, id)
}

func FindTeamByName(name string) (*Team, error) {
	return FindResourceByName[Team](MongoCollectionTeamName, name)
}

func (t *Team) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionTeamName, t.ID, from, to)
}

func (t *Team) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionTeamName, t.ID, version)
}

func (t *Team) migrateSchemaV2() error {
	err := t.renameDbField("users", "user_ids")
	if err != nil {
		return err
	}

	err = t.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (t *Team) Users() []string {
	return t.UserIDs
}

func (t *Team) HasChildren() bool {
	return HasChildRefs(t.UserIDs)
}

func (t *Team) SetUsers(users []string) {
	newUsers := []string{}
	for _, userId := range users {
		// start by checking if that's an actual user
		u, err := FindUserByID(userId)
		if err != nil {
			continue
		}

		// we're adding a new user here
		if !slices.Contains(t.UserIDs, userId) {
			// add team to user
			klog.Infof("Adding user %s to team %s (%s) ...", userId, t.ID, t.Name)
			u.AddTeam(t.String())
		}
		newUsers = append(newUsers, userId)
	}

	// list of users removed
	for _, userId := range t.UserIDs {
		found := false
		for _, uid := range users {
			if userId == uid {
				found = true
				break
			}
		}
		if !found {
			u, err := FindUserByID(userId)
			if err != nil {
				continue
			}
			klog.Infof("Removing user %s from team %s (%s) ...", userId, t.ID, t.Name)
			u.RemoveTeam(t.String())
		}
	}

	t.UserIDs = newUsers
}

func (t *Team) Update(name, desc string, users []string) {
	t.UpdateResourceDefaults(name, desc)
	t.SetUsers(users)
	t.Save()
}

func (t *Team) Save() {
	t.Updated()
	_, err := GetDB().Update(MongoCollectionTeamName, t.ID, t)
	if err != nil {
		klog.Error(err)
	}
}

func (t *Team) Delete() error {
	klog.Debugf("Deleting users team %s", t.String())

	if t.String() == ResourceUnknown {
		return nil
	}

	return GetDB().Delete(MongoCollectionTeamName, t.ID)
}

func (t *Team) Model() sdk.Team {
	return sdk.Team{
		Id:          t.String(),
		Name:        t.Name,
		Description: t.Description,
		Users:       t.UserIDs,
	}
}
