/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package ws

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

func Dial(endpoint, agentType, agentId, agentApiKey string) (*websocket.Conn, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// enforce remote path
	u.Path = WsRouterEndpoint

	klog.Infof("Initiating WebSocket connection to %s", u.String())

	var dialer = websocket.Dialer{
		ReadBufferSize:    WsBufferSize,
		WriteBufferSize:   WsBufferSize,
		EnableCompression: WsCompressionEnabled,
		HandshakeTimeout:  WsHandshakeTimeout * time.Second,
	}

	headers := http.Header{}
	headers.Set(WsHeaderAgentType, agentType)
	headers.Set(WsHeaderAgentId, agentId)
	headers.Set(WsHeaderAgentApiKey, agentApiKey)

	c, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return nil, err
	}

	klog.Infof("WebSocket connection to %s has been successfully established", u.String())

	return c, nil
}
