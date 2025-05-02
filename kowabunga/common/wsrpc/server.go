/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

/*
Heavily inspired from Go's net/rpc and net/rpc/jsonrpc packages

Package wsrpc provides access to the exported methods of an object across an
established WebSocket network connection. A server registers an object, making it visible
as a service with the name of the type of the object.  After registration, exported
methods of the object will be accessible remotely.  A server may register multiple
objects (services) of different types but it is an error to register multiple
objects of the same type.

Only methods that satisfy these criteria will be made available for remote access;
other methods will be ignored:

  - the method's type is exported.
  - the method is exported.
  - the method has two arguments, both exported (or builtin) types.
  - the method's second argument is a pointer.
  - the method has return type error.

In effect, the method must look schematically like

	func (t *T) MethodName(argType T1, replyType *T2) error

where T1 and T2 can be marshaled by encoding/json.

The method's first argument represents the arguments provided by the caller; the
second argument represents the result parameters to be returned to the caller.
The method's return value, if non-nil, is passed back as a string that the client
sees as if created by errors.New.  If an error is returned, the reply parameter
will not be sent back to the client.
*/

package wsrpc

import (
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

type WsRpcServer struct {
	conn       *websocket.Conn
	serviceMap sync.Map // map[string]*service
	methods    []string
	debug      bool
	terminated chan bool
	reason     error
}

func NewWsRpcServer(conn *websocket.Conn, debug bool) *WsRpcServer {
	return &WsRpcServer{
		conn:       conn,
		debug:      debug,
		terminated: make(chan bool, 1),
	}
}

func (server *WsRpcServer) SetWsConnection(conn *websocket.Conn) {
	server.conn = conn
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//   - exported method of exported type
//   - two arguments, both of exported type
//   - the second argument is a pointer
//   - one return value, of type error
//
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *WsRpcServer) Register(rcvr any) error {
	service, err := NewService(rcvr)
	if err != nil {
		klog.Error(err)
		return err
	}

	if _, dup := server.serviceMap.LoadOrStore(service.name, service); dup {
		return errors.New("wsrpc: service already defined: " + service.name)
	}

	for m := range service.method {
		server.methods = append(server.methods, m)
	}

	return nil
}

func (server *WsRpcServer) GetServices() []string {
	services := server.methods
	sort.Strings(services)
	return services
}

func (server *WsRpcServer) handleBinaryMessage(msg []byte) {
	// try to convert received message into RPC request
	req, err := server.getRequest(msg)
	if err != nil {
		klog.Errorf("unable to read RPC request: %s", err)
		return
	}

	// look for registered service method
	svc, mtype, err := server.getServiceMethod(req.ServiceMethod)
	if err != nil {
		klog.Errorf("unable to map RPC service/method: %s", err)
		return
	}

	resp, err := server.call(&req, svc, mtype)
	if err != nil {
		klog.Errorf("unable to execute RPC service/method: %s", err)
		return
	}

	err = server.sendResponse(resp)
	if err != nil {
		klog.Errorf("unable to send RPC response: %s", err)
		if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
			server.terminated <- true
		}
	}
}

func (server *WsRpcServer) getRequest(message []byte) (RpcRequest, error) {
	req := RpcRequest{}

	err := json.Unmarshal(message, &req)
	if err != nil {
		return req, err
	}

	if server.debug {
		klog.Debugf("WsRpcRequest: %#v", req)
	}

	return req, nil
}

func (server *WsRpcServer) getServiceMethod(sm string) (*service, *methodType, error) {
	dot := strings.LastIndex(sm, ".")
	if dot < 0 {
		return nil, nil, errors.New("wsrpc: service/method request ill-formed: " + sm)
	}
	serviceName := sm[:dot]
	methodName := sm[dot+1:]

	// Look up the request.
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		return nil, nil, errors.New("wsrpc: can't find service " + sm)
	}
	svc := svci.(*service)
	mtype := svc.method[methodName]
	if mtype == nil {
		return svc, mtype, errors.New("wsrpc: can't find method " + sm)
	}

	if server.debug {
		klog.Debugf("Service: %+v, MType: %+v", svc, mtype)
	}

	return svc, mtype, nil
}

func (server *WsRpcServer) binConvert(b []byte, x any) error {
	var params = x
	return json.Unmarshal(b, &params)
}

func (server *WsRpcServer) call(req *RpcRequest, svc *service, mtype *methodType) (*RpcResponse, error) {
	// Decode the argument value.
	var argv, replyv reflect.Value
	argIsValue := false // if true, need to indirect before calling.
	if mtype.ArgType.Kind() == reflect.Pointer {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}

	// argv guaranteed to be a pointer now.
	if req.Params == nil {
		return nil, errors.New("wsrpc: missing request params")
	}

	jsonBytes, _ := json.Marshal(req.Params)
	err := server.binConvert(jsonBytes, argv.Interface())
	if err != nil {
		return nil, err
	}

	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}

	function := mtype.method.Func
	returnValues := function.Call([]reflect.Value{svc.rcvr, argv, replyv})

	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(error).Error()
	}

	resp := RpcResponse{
		Id:     req.Id,
		Result: replyv.Interface(),
	}
	if errmsg != "" {
		resp.Error = errmsg
	}

	if server.debug {
		klog.Debugf("WsRpcResponse: %#v", resp)
	}

	return &resp, nil
}

func (server *WsRpcServer) sendResponse(resp *RpcResponse) error {
	msg, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	err = server.conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		return err
	}

	return nil
}

func (server *WsRpcServer) pumpMessages() {
	for {
		// ensure we have a proper connection
		if server.conn == nil {
			server.reason = errors.New("wsrpc: no underlying WebSocket connection")
			server.terminated <- true
			return
		}

		// pump incoming messages
		mt, message, err := server.conn.ReadMessage()
		if err != nil {
			server.reason = err
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				klog.Infof("wsrpc: unexpected remote server connection closure detected")
			}
			server.terminated <- true
			return
		}

		switch mt {
		case websocket.BinaryMessage:
			go server.handleBinaryMessage(message)
		default:
			klog.Infof("Handling other type of message")
		}
	}
}

func (server *WsRpcServer) keepalive() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		klog.Errorf("remote WebSocket disconnection detected")
		ticker.Stop()
		err := server.conn.Close()
		if err != nil {
			klog.Error(err)
		}
	}()

	for {
		<-ticker.C
		err := server.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err != nil {
			return
		}
		err = server.conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			return
		}
	}
}

func (server *WsRpcServer) Listen() error {

	// trap for explicit shutdown request
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// register pong handler, keeping websocket alive
	server.conn.SetPongHandler(func(string) error {
		return server.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// start a background infinite go-routine, handling all messages
	// stop it when underlying connection's broken
	go server.pumpMessages()

	// start a background infinite go-routine, pinging remote peer for keepalive
	go server.keepalive()

	// loop forever until:
	// 1. something's broken on WebSocket connection
	// 2. user requested for explicit shutdown
	select {
	case <-server.terminated:
		klog.Infof("WebSocket's broken: %s", server.reason)
		return server.reason
	case <-quit:
		klog.Infof("Explicit interruption's request, closing WebSocket connection ...")

		// Cleanly close the connection by sending a close message and then
		// waiting (with timeout) for the server to close the connection.

		// send proper close notification
		err := server.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			klog.Errorf("Unable to properly shutdown connection: %s", err)
			return nil // still, explicit request to shutdown
		}

		// wait (with timeout) for server to close the connection
		select {
		case <-server.terminated:
		case <-time.After(WsCloseRequestTimeout):
		}

		return nil
	}
}
