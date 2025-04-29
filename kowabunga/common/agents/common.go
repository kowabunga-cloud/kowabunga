/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

import (
	"os"
	"sync"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/wsrpc"
)

type KowabungaAgent struct {
	id                string
	kind              string
	endpoint          string
	apikey            string
	reconnectInterval int
	rpcServer         *wsrpc.WsRpcServer
	wg                *sync.WaitGroup
	shutdown          chan os.Signal
	PostFlight        func()
}

func NewKowabungaAgent(id, kind, endpoint, apikey string) *KowabungaAgent {
	agent := KowabungaAgent{
		id:                id,
		kind:              kind,
		endpoint:          endpoint,
		apikey:            apikey,
		reconnectInterval: 0,
		rpcServer:         wsrpc.NewWsRpcServer(nil, false), // initialize non-connected RPC server
		wg:                &sync.WaitGroup{},
		shutdown:          make(chan os.Signal, 1),
		PostFlight:        nil,
	}

	return &agent
}

func (agent *KowabungaAgent) RpcServer() *wsrpc.WsRpcServer {
	return agent.rpcServer
}

func (agent *KowabungaAgent) RegisterServices(services ...any) error {
	// register all remote procedure call services and associated methods
	for _, s := range services {
		err := agent.rpcServer.Register(s)
		if err != nil {
			klog.Errorf("Unable to register service %s: %s", s, err)
			return err
		}
	}

	klog.Debug("Registered RPC services:")
	methods := agent.rpcServer.GetServices()
	for _, m := range methods {
		klog.Debugf(" - %s.%s()", agent.kind, m)
	}

	return nil
}

// all agents must implement the Capabilities() method for server to discover features
type CapabilitiesArgs struct{}
type CapabilitiesReply struct {
	Version string
	Methods []string
}
