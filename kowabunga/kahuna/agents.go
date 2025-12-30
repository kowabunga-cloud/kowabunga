/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"sync"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/agents"
	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/common/wsrpc"
)

type KowabungaAgent struct {
	ID          string
	Client      *wsrpc.WsRpcClient
	Type        string
	IsConnected bool
	Interrupted chan bool
	Version     string
	Methods     []string
}

func (agent *KowabungaAgent) WatchKeepalive() {
	<-agent.Client.Terminated
	klog.Infof("Disconnection of %s agent %s has been detected. Unregistering ...", agent.Type, agent.ID)
	agent.Interrupted <- true
	UnregisterAgent(agent.ID)
}

// agents singleton
var agentsLock = &sync.Mutex{}
var kAgents map[string]*KowabungaAgent

func GetAgents() map[string]*KowabungaAgent {
	if kAgents == nil {
		agentsLock.Lock()
		defer agentsLock.Unlock()
		klog.Debugf("Creating Kowabunga Agents map")
		kAgents = map[string]*KowabungaAgent{}
	}

	return kAgents
}

func DisconnectAgent(agentId string) {
	ag := GetAgent(agentId)
	if ag == nil {
		return
	}
	ag.Interrupted <- true
	ag.Client.Terminated <- true
}

func GetAgent(agentId string) *KowabungaAgent {
	return GetAgents()[agentId]
}

func GetEligibleAgent(candidateAgents []string, method string) *KowabungaAgent {
	candidates := []*KowabungaAgent{}

	// build list of agents capable of addressing requested method
	for _, ag := range GetAgents() {
		// discard agents not associated with the RPC caller
		if !slices.Contains(candidateAgents, ag.ID) {
			continue
		}

		// discard agent with non-implemented RPC method
		if !slices.Contains(ag.Methods, method) {
			continue
		}

		// discard offline agents
		if !ag.IsConnected || ag.Client == nil {
			continue
		}

		candidates = append(candidates, ag)
	}

	if len(candidates) == 0 {
		return nil
	}

	// randomly address one agent
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidates))))
	if err != nil {
		return nil
	}

	return candidates[n.Int64()]
}

func RegisterAgent(agentType, agentId string, client *wsrpc.WsRpcClient) error {
	// check for an already registered agent
	ag := GetAgent(agentId)
	if ag != nil {
		// agent already exists, check for status
		if ag.IsConnected {
			// close previous session
			klog.Infof("A previous agent WebSocket connection from %s was referenced. Closing ...", agentId)
			err := ag.Client.Close()
			if err != nil {
				klog.Error(err)
			}
		}
		ag.Client = client
		return nil
	}

	ag = &KowabungaAgent{
		ID:          agentId,
		Client:      client,
		Type:        agentType,
		IsConnected: true,
		Interrupted: make(chan bool, 1),
	}

	// discover agent's RPC capabilities
	args := agents.CapabilitiesArgs{}
	var reply agents.CapabilitiesReply
	method := fmt.Sprintf("%s.Capabilities", ag.Type)
	err := ag.Client.Call(method, args, &reply)
	if err != nil {
		klog.Errorf("Unable to call remote Capabilities() RPC. Is it a legit agent ??")
		return err
	}
	ag.Version = reply.Version
	ag.Methods = reply.Methods

	klog.Infof("Registering new %s agent %s WebSocket connection ...", agentType, agentId)
	agentsLock.Lock()
	GetAgents()[agentId] = ag
	agentsLock.Unlock()
	go ag.WatchKeepalive()

	switch ag.Type {
	case common.KowabungaKiwiAgent:
		kiwis := FindKiwis()
		for _, k := range kiwis {
			if slices.Contains(k.AgentIDs, agentId) {
				err := k.Reload()
				if err != nil {
					return err
				}
			}
		}
	case common.KowabungaKaktusAgent:
		kaktuses := FindKaktuses()
		for _, k := range kaktuses {
			if slices.Contains(k.Agents(), agentId) {
				go k.Scan()
				break
			}
		}

		pools := FindStoragePools()
		for _, p := range pools {
			if slices.Contains(p.Agents(), agentId) {
				go p.Scan()
				break
			}
		}
	case common.KowabungaControllerAgent:
		args := agents.KontrollerReloadArgs{}
		var reply agents.KontrollerReloadReply
		err = RPC([]string{agentId}, "Reload", args, &reply)
		if err != nil {
			return err
		}
	}

	return nil
}

func UnregisterAgent(agentId string) {
	agentsLock.Lock()
	delete(GetAgents(), agentId)
	agentsLock.Unlock()
}

func VerifyAgents(candidates []string, kind string) []string {
	agents := []string{}

	// add eligible agents
	for _, agentId := range candidates {
		a, err := FindAgentByID(agentId)
		if err != nil {
			klog.Errorf("invalid agent ID: %s (%v)", agentId, err)
			continue
		}
		if a.Type != kind {
			klog.Errorf("invalid agent type")
			continue
		}
		agents = append(agents, agentId)
	}

	return agents
}

func RPC(candidateAgents []string, method string, args, reply any) error {
	ag := GetEligibleAgent(candidateAgents, method)
	if ag == nil {
		return fmt.Errorf("RPC: can't find any eligible agent to perform such request")
	}

	methodName := fmt.Sprintf("%s.%s", ag.Type, method)
	return ag.Client.Call(methodName, args, reply)
}
