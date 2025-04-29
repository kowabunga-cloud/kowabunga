/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

import (
	"os/signal"
	"syscall"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/ws"
)

const (
	WsReconnectMinInterval = 5  // seconds
	WsReconnectMaxInterval = 30 // seconds
)

func (agent *KowabungaAgent) waitForReconnection() bool {
	if agent.reconnectInterval > WsReconnectMaxInterval {
		agent.reconnectInterval = WsReconnectMinInterval
	} else {
		agent.reconnectInterval += WsReconnectMinInterval
	}

	klog.Infof("Waiting for %d seconds before trying to re-establish connection ...", agent.reconnectInterval)

	select {
	case <-agent.shutdown:
		klog.Infof("Explicit shutdown has been requested ...")
		if agent.PostFlight != nil {
			agent.PostFlight()
		}
		return false
	case <-time.After(time.Duration(agent.reconnectInterval) * time.Second):
		return true
	}
}

func (agent *KowabungaAgent) connect() error {
	defer agent.wg.Done()

	c, err := ws.Dial(agent.endpoint, agent.kind, agent.id, agent.apikey)
	if err != nil {
		klog.Errorf("WS connection failed: %s", err)
		return err
	}

	// reset interval counter
	agent.reconnectInterval = 0

	agent.rpcServer.SetWsConnection(c)
	return agent.rpcServer.Listen()
}

func (agent *KowabungaAgent) Run() {
	// trap for explicit shutdown request
	signal.Notify(agent.shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Try forever to establish WebSocket agent connection to Kowabunga remote orchestrator.
	// Anything happens on remote-side, we'll keep on trying unless explicitely requested to stop
	// through user signal interrupt
	for {
		agent.wg.Add(1)

		// establish a new WebSocket connection
		err := agent.connect()

		// explicit stop has been requested, let's quit
		if err == nil {
			break
		}

		// wait for grace period before trying to re-establish connection
		// or quit if interrupted
		if !agent.waitForReconnection() {
			break
		}
	}
	agent.wg.Wait()
}
