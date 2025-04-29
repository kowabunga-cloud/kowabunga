/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func wsCheckOrigin(r *http.Request) bool {
	return true // bypass cross-origin checks
}

func ServerConnectionUpgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:    WsBufferSize,
		WriteBufferSize:   WsBufferSize,
		EnableCompression: WsCompressionEnabled,
		HandshakeTimeout:  WsHandshakeTimeout * time.Second,
		CheckOrigin:       wsCheckOrigin,
	}

	return upgrader.Upgrade(w, r, nil)
}
