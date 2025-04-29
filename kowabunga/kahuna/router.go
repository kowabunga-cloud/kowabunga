/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/ws"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/wsrpc"

	"github.com/gorilla/mux"
)

const (
	HttpHeaderAuthApiKey       = "X-API-Key"
	HttpHeaderAuthorization    = "Authorization"
	HttpHeaderAuthBearerPrefix = "Bearer "
	HttpApiMethodAll           = "*"
)

type ApiOperation struct {
	Route  string
	Method string
}

var noAuthApiOperations = []string{
	"Login",
	"ResetPassword",
}

var userAllowedRoutes = []ApiOperation{
	{
		Route:  "/logout",
		Method: "POST",
	},
	{
		Route:  "/user/{userId}.*",
		Method: HttpApiMethodAll,
	},
	{
		Route:  "/project$",
		Method: "GET",
	},
	{
		Route:  "/region$",
		Method: "GET",
	},
	{
		Route:  "/region/[A-Za-z0-9]*$",
		Method: "GET",
	},
	{
		Route:  "/zone$",
		Method: "GET",
	},
	{
		Route:  "/zone/[A-Za-z0-9]*$",
		Method: "GET",
	},
}

var projectAdminAllowedRoutes = []ApiOperation{
	{
		Route:  "/project$",
		Method: HttpApiMethodAll,
	},
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	v := fmt.Sprintf("%s (%s)\n", version, codename)
	_, err := w.Write([]byte(v))
	if err != nil {
		klog.Error(err)
	}
}

func htmlResponse(param string) string {
	return fmt.Sprintf(
		`<html>
                   <head><title>Kowabunga</title></head>
                   <body>
                     <p>%s</p>
                   </body>
                 </html>`,
		param)
}

func userRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("user")
	userToken := r.URL.Query().Get("token")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	user, err := FindUserByID(userId)
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(htmlResponse("Unable to find user")))
		return
	}

	if userToken != user.RegistrationToken {
		klog.Error(err)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(htmlResponse("Invalid user registration token")))
		return
	}

	// all good, verified
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(htmlResponse("User is now fully registered")))
	user.Enable()
}

func userPasswordRenewalHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("user")
	userToken := r.URL.Query().Get("token")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	user, err := FindUserByID(userId)
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(htmlResponse("Unable to find user")))
		return
	}

	if userToken != user.PasswordRenewalToken {
		klog.Error(err)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(htmlResponse("Invalid user password renewal token")))
		return
	}

	// all good, verified
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(htmlResponse("A new user password has been issued")))
	err = user.ResetPassword()
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(htmlResponse("Unable to handle password renewal request")))
		return
	}
}

func metadataHandler(w http.ResponseWriter, r *http.Request) {
	sourceHeader := r.Header.Get(common.HttpHeaderKowabungaSourceIP)
	src := strings.Split(sourceHeader, ":")
	meta, err := GetInstanceMetadata(src[0], r.Header.Get(common.HttpHeaderKowabungaInstanceID))
	if err != nil {
		klog.Error(err)
		return
	}

	err = json.NewEncoder(w).Encode(meta)
	if err != nil {
		klog.Error(err)
	}
}

func loggingMiddleware(next http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		klog.Debugf("%s %s %s %s", r.Method, r.RequestURI, name, time.Since(start))
	})
}

func ctxSet(ctx context.Context, key, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

func ctxGet(ctx context.Context, key string) any {
	return ctx.Value(key)
}

const (
	HttpRequestContextUserId           = "userId"
	HttpRequestContextSuperAdminRole   = "superAdminRole"
	HttpRequestContextProjectAdminRole = "projectAdminRole"
	HttpRequestContextAuthMethod       = "authMethod"
)

func ctxSetUserId(ctx context.Context, value string) context.Context {
	return ctxSet(ctx, HttpRequestContextUserId, value)
}

func ctxGetUserId(ctx context.Context) string {
	value := ctxGet(ctx, HttpRequestContextUserId)
	if value == nil {
		return ""
	}
	return value.(string)
}

func ctxSetAuthMethod(ctx context.Context, method string) context.Context {
	return ctxSet(ctx, HttpRequestContextAuthMethod, method)
}

func ctxGetAuthMethod(ctx context.Context) string {
	value := ctxGet(ctx, HttpRequestContextAuthMethod)
	if value == nil {
		return ""
	}
	return value.(string)
}

func ctxSetSuperAdminRole(ctx context.Context) context.Context {
	return ctxSet(ctx, HttpRequestContextSuperAdminRole, true)
}

func ctxGetSuperAdminRole(ctx context.Context) bool {
	value := ctxGet(ctx, HttpRequestContextSuperAdminRole)
	return value != nil
}

func ctxSetProjectAdminRole(ctx context.Context) context.Context {
	return ctxSet(ctx, HttpRequestContextProjectAdminRole, true)
}

func ctxGetProjectAdminRole(ctx context.Context) bool {
	value := ctxGet(ctx, HttpRequestContextProjectAdminRole)
	return value != nil
}

func reqIsAuthenticated(r *http.Request) (*http.Request, bool) {
	ctx := r.Context()

	// start with server-to-server API-key based authentication
	apikey := r.Header.Get(HttpHeaderAuthApiKey)
	if apikey != "" {
		// check if passed API key correspond to Kowabunga master one
		if apikey == GetCfg().Global.APIKey {
			klog.Debugf("API-key based authentication")
			ctx = ctxSetSuperAdminRole(ctx)
			return r.WithContext(ctx), true
		}

		// check for regular user API key
		tokens := FindTokens()
		for _, t := range tokens {
			// currently no better way than to verify all registered API keys ... ;-(
			if t.ParentType != TokenParentTypeUser {
				continue
			}

			err := t.Verify(apikey)
			if err != nil {
				continue
			}

			// we found the right token, let's verify validity
			if t.HasExpired() {
				return nil, false
			}

			u, err := t.User()
			if err != nil {
				return nil, false
			}

			ctx = ctxSetAuthMethod(ctx, HttpHeaderAuthApiKey)
			ctx = ctxSetUserId(ctx, u.String())
			if u.IsSuperAdmin() {
				ctx = ctxSetSuperAdminRole(ctx)
			}
			if u.IsProjectAdmin() {
				ctx = ctxSetProjectAdminRole(ctx)
			}
			klog.Debugf("API-key based authentication")
			return r.WithContext(ctx), true
		}

		return nil, false
	}

	// failover, JWT-based auth
	authHeader := r.Header.Get(HttpHeaderAuthorization)
	if authHeader == "" {
		// No authentication scheme header in HTTP request
		return nil, false
	}

	jwtToken := authHeader[len(HttpHeaderAuthBearerPrefix):]
	_, uid, err := VerifyJwt(jwtToken)
	if err != nil {
		klog.Error(err)
		return nil, false
	}

	klog.Debugf("JWT based authentication")
	ctx = ctxSetAuthMethod(ctx, HttpHeaderAuthorization)
	ctx = ctxSetUserId(ctx, uid)

	u, err := FindUserByID(uid)
	if err != nil {
		return nil, false
	}
	if u.IsSuperAdmin() {
		ctx = ctxSetSuperAdminRole(ctx)
	}
	if u.IsProjectAdmin() {
		ctx = ctxSetProjectAdminRole(ctx)
	}

	return r.WithContext(ctx), true
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated
		rAuth, ok := reqIsAuthenticated(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Call the next middleware function or final handler
		next.ServeHTTP(w, rAuth)
	})
}

func validMethod(method string, op ApiOperation) bool {
	if op.Method == HttpApiMethodAll {
		return true
	}
	return strings.EqualFold(method, op.Method)
}

func reqIsAuthorized(r *http.Request) bool {
	ctx := r.Context()

	// check for almighty god rights
	superAdmin := ctxGetSuperAdminRole(ctx)
	if superAdmin {
		return true
	}

	// check project admin rights
	projectAdmin := ctxGetProjectAdminRole(ctx)
	if projectAdmin {
		for _, route := range projectAdminAllowedRoutes {
			match, _ := regexp.MatchString(SdkBaseRoute+route.Route, r.RequestURI)
			if match && validMethod(r.Method, route) {
				return true
			}
		}
	}

	// check for user-allowed routes
	userId := ctxGetUserId(ctx)
	if userId != "" {
		for _, route := range userAllowedRoutes {
			rt := SdkBaseRoute + strings.Replace(route.Route, "{userId}", userId, -1)
			match, _ := regexp.MatchString(rt, r.RequestURI)
			if match && validMethod(r.Method, route) {
				return true
			}
		}
	}

	// checked for cached user resources mapping
	userAllowedResources := []string{}
	err := GetCache().Get(CacheNsUserResources, userId, &userAllowedResources)
	if err != nil {
		// cache miss: crawl over DB

		// check for any other route: list all resources from projects where user's team is allowed to
		// and check whether omne of the resource IDs match the request URI
		u, err := FindUserByID(userId)
		if err != nil {
			return false
		}
		for _, prj := range FindProjects() {
			for _, teamId := range u.Teams() {
				if !slices.Contains(prj.TeamIDs, teamId) {
					continue
				}
				userAllowedResources = append(userAllowedResources, prj.Resources()...)
			}
		}

		// store in cache
		GetCache().Set(CacheNsUserResources, userId, userAllowedResources)
	}

	// Check if request url contains one of the resource IDs from the projects user's part of
	for _, res := range userAllowedResources {
		if strings.Contains(r.RequestURI, res) {
			return true
		}
	}

	// everything else is denied
	return false
}

func authorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is allowed to request
		if !reqIsAuthorized(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		// Call the next middleware function or final handler
		next.ServeHTTP(w, r)
	})
}

func verifyAgentHeaders(w http.ResponseWriter, r *http.Request) (string, string, error) {
	// check for valid agent connection
	agentType := r.Header.Get(ws.WsHeaderAgentType)
	if agentType == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return agentType, "", fmt.Errorf("Unspecified Kowabunga agent type")
	}

	if !slices.Contains(common.SupportedAgents(), agentType) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return agentType, "", fmt.Errorf("Unsupported Kowabunga agent type")
	}

	// check for valid agent ID
	agentId := r.Header.Get(ws.WsHeaderAgentId)
	if agentId == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return agentType, agentId, fmt.Errorf("Unspecified Kowabunga agent ID")
	}

	ag, err := FindAgentByID(agentId)
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return agentType, agentId, fmt.Errorf("Unsupported Kowabunga agent ID")
	}

	// check for agent API key authentication
	apiKey := r.Header.Get(ws.WsHeaderAgentApiKey)
	if apiKey == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return agentType, agentId, fmt.Errorf("Unspecified Kowabunga agent API key")
	}

	token, err := ag.Token()
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return agentType, agentId, fmt.Errorf("No api key token associated to agent")
	}

	err = token.Verify(apiKey)
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return agentType, agentId, fmt.Errorf("API key does not match")
	}

	// verify token's validity
	if token.HasExpired() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return agentType, agentId, fmt.Errorf("API key has expired")
	}

	return agentType, agentId, nil
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	agentType, agentId, err := verifyAgentHeaders(w, r)
	if err != nil {
		klog.Errorf("ws headers: %v", err)
		return
	}

	c, err := ws.ServerConnectionUpgrade(w, r)
	if err != nil {
		klog.Errorf("ws upgrade: %s", err)
		return
	}

	client := wsrpc.NewWsRpcClient(c, false)
	err = RegisterAgent(agentType, agentId, client)
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

func NewRouter(ke *KahunaEngine) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// add all sub-routes from SDK services
	for _, api := range ke.ApiRouters {
		for name, route := range api.Routes() {
			var handler http.Handler
			handler = route.HandlerFunc

			//
			// WARNING: reverse-order logic in middlewares queueing
			//

			if !slices.Contains(noAuthApiOperations, name) {
				// authorization middelware
				handler = authorizationMiddleware(handler)

				// authentication middelware
				handler = authenticationMiddleware(handler)
			}

			// logging middleware
			handler = loggingMiddleware(handler, name)

			router.
				Methods(route.Method).
				Path(route.Pattern).
				Name(name).
				Handler(handler)
		}
	}

	// extra endpoint routes
	router.HandleFunc("/confirm", userRegistrationHandler)
	router.HandleFunc("/confirmForgotPassword", userPasswordRenewalHandler)
	router.HandleFunc("/version", versionHandler)
	router.HandleFunc("/latest/meta-data", metadataHandler)
	router.Handle("/metrics", ke.Exporter.HttpHandler())
	router.HandleFunc(ws.WsRouterEndpoint, wsHandler)

	return router
}
