/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package wsrpc

import (
	"time"
)

const (
	WsCloseRequestTimeout  = 60 * time.Second
	WsCloseResponseTimeout = 3600 * time.Second
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type RpcRequest struct {
	Id            uint64 `json:"id"`
	ServiceMethod string `json:"method"`
	Params        any    `json:"params"`
}

type RpcResponse struct {
	Id     uint64 `json:"id"`
	Result any    `json:"result"`
	Error  any    `json:"error"`
}
