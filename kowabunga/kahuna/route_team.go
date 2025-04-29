/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewTeamRouter() sdk.Router {
	return sdk.NewTeamAPIController(&TeamService{})
}

type TeamService struct{}

func (s *TeamService) CreateTeam(ctx context.Context, team sdk.Team) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("team", team))

	// check for params
	if team.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure team does not already exists
	_, err := FindTeamByName(team.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create team
	g, err := NewTeam(team.Name, team.Description, team.Users)
	if err != nil {
		return HttpServerError(err)
	}

	payload := g.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *TeamService) DeleteTeam(ctx context.Context, teamId string) (sdk.ImplResponse, error) {
	// ensure user exists
	g, err := FindTeamByID(teamId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if team still has children referenced
	if g.HasChildren() {
		return HttpConflict(err)
	}

	// remove team
	err = g.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *TeamService) ListTeams(ctx context.Context) (sdk.ImplResponse, error) {
	teams := FindTeams()
	var payload []string
	for _, g := range teams {
		payload = append(payload, g.String())
	}

	return HttpOK(payload)
}

func (s *TeamService) ReadTeam(ctx context.Context, teamId string) (sdk.ImplResponse, error) {
	g, err := FindTeamByID(teamId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := g.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *TeamService) UpdateTeam(ctx context.Context, teamId string, team sdk.Team) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("teamId", teamId), RA("team", team))

	// check for params
	if team.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure team exists
	g, err := FindTeamByID(teamId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update team
	g.Update(team.Name, team.Description, team.Users)

	payload := g.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
