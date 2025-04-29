/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package ws

const (
	WsCompressionEnabled = false
	WsHandshakeTimeout   = 45   // seconds
	WsBufferSize         = 8192 // 8kiB
	WsRouterEndpoint     = "/ws"
	WsHeaderAgentType    = "X-WS-Kowabunga-Agent"
	WsHeaderAgentId      = "X-WS-Kowabunga-Id"
	WsHeaderAgentApiKey  = "X-WS-Kowabunga-Api-Key"
)
