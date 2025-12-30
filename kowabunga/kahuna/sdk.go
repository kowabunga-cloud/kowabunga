/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	SdkBaseRoute = "/api/v1"
)

type RequestArg struct {
	Name  string
	Value interface{}
}

func (a *RequestArg) String() string {
	return fmt.Sprintf("%s:%#v", a.Name, a.Value)
}

func RA(name string, value interface{}) RequestArg {
	return RequestArg{
		Name:  name,
		Value: value,
	}
}

func LogHttpRequest(args ...RequestArg) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	funcname := runtime.FuncForPC(pc).Name()
	fn := funcname[strings.LastIndex(funcname, ".")+1:]
	var params string
	for _, a := range args {
		params += fmt.Sprintf("%s ", a.String())
	}
	msg := fmt.Sprintf("%s() request params - %s", fn, params)
	klog.Debug(msg)
}

func LogHttpResponse(body interface{}) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	funcname := runtime.FuncForPC(pc).Name()
	fn := funcname[strings.LastIndex(funcname, ".")+1:]
	msg := fmt.Sprintf("%s() response body - %+v", fn, body)
	klog.Debug(msg)
}

func HttpOK(body interface{}) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusOK, body), nil
}

func HttpUnauthorized(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusUnauthorized, sdk.ApiErrorUnauthorized{}), err
}

func HttpForbidden(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusForbidden, sdk.ApiErrorForbidden{}), err
}

func HttpCreated(body interface{}) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusCreated, body), nil
}

func HttpCreatedNoContent() (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusNoContent, nil), nil
}

func HttpBadParams(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusBadRequest, sdk.ApiErrorBadRequest{}), err
}

func HttpNotFound(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusNotFound, sdk.ApiErrorNotFound{}), err
}

func HttpConflict(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusConflict, sdk.ApiErrorConflict{}), err
}

func HttpServerError(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusUnprocessableEntity, sdk.ApiErrorConflict{}), err
}

func HttpQuota(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusInsufficientStorage, sdk.ApiErrorInsufficientResource{}), err
}

func HttpNotImplemented(err error) (sdk.ImplResponse, error) {
	return sdk.Response(http.StatusNotImplemented, nil), err
}
