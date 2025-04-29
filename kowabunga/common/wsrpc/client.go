/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package wsrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"

	"github.com/gorilla/websocket"
)

type WsRpcClient struct {
	mutex      sync.Mutex // protects following
	seq        uint64
	conn       *websocket.Conn
	response   chan []byte
	Terminated chan bool
	debug      bool
}

func NewWsRpcClient(conn *websocket.Conn, debug bool) *WsRpcClient {
	client := WsRpcClient{
		conn:       conn,
		debug:      debug,
		response:   make(chan []byte, 1),
		Terminated: make(chan bool, 1),
	}

	// register pong handler, keeping websocket alive
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// start a background infinite go-routine, handling all messages
	// stop it when underlying connection's broken
	go client.pumpMessages()

	// start a background infinite go-routine, pinging remote peer for keepalive
	go client.keepalive()

	return &client
}

func (client *WsRpcClient) pumpMessages() {
	for {
		// ensure we have a proper connection
		if client.conn == nil {
			klog.Errorf("wsrpc: no underlying WebSocket connection")
			client.Terminated <- true
			return
		}

		// pump incoming messages
		mt, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				klog.Infof("wsrpc: unexpected remote client connection closure detected")
			}
			errC := client.conn.Close()
			if errC != nil {
				klog.Errorf("wsrpc: unable to properly close connection: %v", err)
			}
			client.Terminated <- true
			return
		}

		switch mt {
		case websocket.BinaryMessage:
			// deliver pumped message to channel, ready to be processed by caller
			client.response <- message
		default:
			klog.Infof("Handling other type of message: %d", mt)
		}
	}
}

func (client *WsRpcClient) keepalive() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		klog.Errorf("remote WebSocket disconnection detected")
		ticker.Stop()
		err := client.conn.Close()
		if err != nil {
			klog.Errorf("wsrpc: unable to properly close connection: %v", err)
		}
		client.Terminated <- true
	}()

	for {
		<-ticker.C
		err := client.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err != nil {
			return
		}
		err = client.conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			return
		}
	}
}

func (client *WsRpcClient) Close() error {
	return client.conn.Close()
}

func (client *WsRpcClient) newRequest(serviceMethod string, args any) *RpcRequest {
	seq := client.seq
	client.seq++

	req := RpcRequest{
		Id:            seq,
		ServiceMethod: serviceMethod,
		Params:        args,
	}

	if client.debug {
		klog.Debugf("WsRpcRequest: %#v", req)
	}

	return &req
}

func (client *WsRpcClient) sendRequest(req *RpcRequest) error {
	msg, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// write RPC request to server
	err = client.conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		client.Terminated <- true
		return err
	}

	return nil
}

func (client *WsRpcClient) getResponse(reply any) (*RpcResponse, error) {

	// dequeue (with timeout) messages from handler, whichever comes first
	select {
	case message := <-client.response:
		resp := RpcResponse{
			Result: reply, // pointer to reply, for unmarshalling into
		}

		// read RPC reply from server
		err := json.Unmarshal(message, &resp)
		if err != nil {
			return &resp, err
		}

		if client.debug {
			klog.Debugf("WsRpcResponse: %#v", resp)
		}

		return &resp, nil
	case <-time.After(WsCloseResponseTimeout):
		return nil, errors.New("wsrpc: timeout")
	}
}

func (client *WsRpcClient) Call(serviceMethod string, args any, reply any) error {

	// prevent concurential calls from happening on the same WebSocket transport layer
	client.mutex.Lock()
	defer client.mutex.Unlock()

	req := client.newRequest(serviceMethod, args)
	err := client.sendRequest(req)
	if err != nil {
		klog.Errorf("unable to send RPC request: %s", err)
		return err
	}

	resp, err := client.getResponse(reply)
	if err != nil {
		klog.Errorf("unable to read RPC response: %s", err)
		return err
	}

	if resp.Id != req.Id {
		err := errors.New("wsrpc: mismatch between RPC request and response IDs")
		klog.Error(err)
		return err
	}

	if resp.Error != nil {
		err := fmt.Errorf("wsrpc: %s() error (%v)", serviceMethod, resp.Error)
		klog.Error(err)
		return err
	}

	return nil
}
