/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewAgentRouter() sdk.Router {
	return sdk.NewAgentAPIController(&AgentService{})
}

type AgentService struct{}

func (s *AgentService) CreateAgent(ctx context.Context, agent sdk.Agent) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("agent", agent))

	// check for params
	if agent.Name == "" {
		return HttpBadParams(nil)
	}

	// ensure agent does not already exists
	_, err := FindAgentByName(agent.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create agent
	a, err := NewAgent(agent.Name, agent.Description, agent.Type)
	if err != nil {
		return HttpServerError(err)
	}

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *AgentService) DeleteAgent(ctx context.Context, agentId string) (sdk.ImplResponse, error) {
	// ensure agent exists
	a, err := FindAgentByID(agentId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove agent
	err = a.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	// disconnect any live agent WebSocket, if any
	DisconnectAgent(agentId)

	return HttpOK(nil)
}

func (s *AgentService) ListAgents(ctx context.Context) (sdk.ImplResponse, error) {
	agents := FindAgents()
	var payload []string
	for _, a := range agents {
		payload = append(payload, a.String())
	}

	return HttpOK(payload)
}

func (s *AgentService) ReadAgent(ctx context.Context, agentId string) (sdk.ImplResponse, error) {
	a, err := FindAgentByID(agentId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *AgentService) SetAgentApiToken(ctx context.Context, agentId string, expire bool, expirationDate string) (sdk.ImplResponse, error) {
	a, err := FindAgentByID(agentId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if agent already has a registered token
	var t *Token

	tokenName := fmt.Sprintf("%s-api-key", a.Name)
	t, err = FindTokenByName(tokenName)
	if err != nil {
		// can't find any token, will create a new one
		t, err = NewAgentToken(agentId, tokenName, "", expire, expirationDate)
		if err != nil {
			return HttpServerError(err)
		}
	}

	// update token's expiration date, if any
	err = t.Update(tokenName, "", expire, expirationDate)
	if err != nil {
		return HttpServerError(err)
	}

	_, err = t.SetNewApiKey(true)
	if err != nil {
		return HttpServerError(err)
	}

	// disconnect any live agent WebSocket, if any
	DisconnectAgent(agentId)

	payload := t.Model()
	return HttpOK(payload)
}

func (s *AgentService) UpdateAgent(ctx context.Context, agentId string, agent sdk.Agent) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("agentId", agentId), RA("agent", agent))

	// check for params
	if agent.Name == "" && agent.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure token exists
	a, err := FindAgentByID(agentId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update agent
	a.Update(agent.Name, agent.Description)

	payload := a.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
