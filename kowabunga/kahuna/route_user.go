/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewUserRouter() sdk.Router {
	return sdk.NewUserAPIController(&UserService{})
}

type UserService struct{}

func (s *UserService) Login(ctx context.Context, userCredentials sdk.UserCredentials) (sdk.ImplResponse, error) {
	klog.Debugf("Login attempt from %s ...", userCredentials.Email)

	if userCredentials.Email == "" || userCredentials.Password == "" {
		return HttpBadParams(nil)
	}

	// ensure user exists
	u, err := FindUserByEmail(userCredentials.Email)
	if err != nil {
		klog.Error(err)
		return HttpNotFound(err)
	}

	// ensure user has finalized account creation
	if !u.Enabled {
		return HttpUnauthorized(err)
	}

	// check if password matches registered one
	err = u.Verify(userCredentials.Password)
	if err != nil {
		klog.Error(err)
		return HttpUnauthorized(err)
	}

	// let's go with JWT
	jwt, err := u.JwtSession()
	if err != nil {
		klog.Error(err)
		return HttpServerError(err)
	}

	userCredentials.Jwt = jwt
	return HttpCreated(userCredentials)
}

func (s *UserService) Logout(ctx context.Context) (sdk.ImplResponse, error) {
	if ctxGetAuthMethod(ctx) != HttpHeaderAuthorization {
		klog.Warningf("Attempting to logout from a non JWT-based session; pointless")
		return HttpCreatedNoContent() // not an error
	}

	userId := ctxGetUserId(ctx)
	klog.Debugf("Trying to logout user %s ...", userId)

	user, err := FindUserByID(userId)
	if err != nil {
		return HttpServerError(err)
	}

	klog.Debugf("Invalidating JWTtoken for user %s", userId)
	user.JWT = ""
	user.Save()

	return HttpCreatedNoContent()
}

func (s *UserService) CreateUser(ctx context.Context, user sdk.User) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("user", user))

	// check for params
	if user.Name == "" || user.Email == "" {
		return HttpBadParams(nil)
	}

	// ensure user does not already exists
	_, err := FindUserByName(user.Name)
	if err == nil {
		return HttpConflict(err)
	}

	// create user
	u, err := NewUser(user.Name, user.Description, user.Email, user.Role, user.Notifications)
	if err != nil {
		return HttpServerError(err)
	}

	payload := u.Model()
	LogHttpResponse(payload)
	return HttpCreated(payload)
}

func (s *UserService) DeleteUser(ctx context.Context, userId string) (sdk.ImplResponse, error) {
	// ensure user exists
	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove user
	err = u.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *UserService) ListUsers(ctx context.Context) (sdk.ImplResponse, error) {
	users := FindUsers()
	var payload []string
	for _, u := range users {
		payload = append(payload, u.String())
	}

	return HttpOK(payload)
}

func (s *UserService) ReadUser(ctx context.Context, userId string) (sdk.ImplResponse, error) {
	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := u.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *UserService) SetUserPassword(ctx context.Context, userId string, password sdk.Password) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("userId", userId))

	// check for params
	if password.Value == "" {
		return HttpBadParams(nil)
	}

	// ensure user exists
	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update user password
	err = u.UpdatePassword(password.Value)
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *UserService) ResetPassword(ctx context.Context, email sdk.UserEmail) (sdk.ImplResponse, error) {
	u, err := FindUserByEmail(email.Email)
	if err != nil {
		return HttpServerError(err)
	}

	// reset user password
	err = u.ResetPasswordRequest()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *UserService) ResetUserPassword(ctx context.Context, userId string) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("userId", userId))

	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	// reset user password
	err = u.ResetPasswordRequest()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *UserService) SetUserApiToken(ctx context.Context, userId string, expire bool, expirationDate string) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("userId", userId), RA("expire", expire), RA("expirationDate", expirationDate))

	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	// check if user already has a registered token
	var t *Token

	tokenName := fmt.Sprintf("%s-api-key", u.Name)
	t, err = FindTokenByName(tokenName)
	if err != nil {
		// can't find any token, will create a new one
		t, err = NewUserToken(userId, tokenName, "", expire, expirationDate)
		if err != nil {
			return HttpServerError(err)
		}
	}

	// update token's expiration date, if any
	err = t.Update(tokenName, "", expire, expirationDate)
	if err != nil {
		return HttpServerError(err)
	}

	_, err = t.SetNewApiKey(true)
	if err != nil {
		return HttpServerError(err)
	}

	payload := t.Model()
	return HttpOK(payload)
}

func (s *UserService) UpdateUser(ctx context.Context, userId string, user sdk.User) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("userId", userId), RA("user", user))

	// check for params
	if user.Name == "" && user.Email == "" && user.Role == "" {
		return HttpBadParams(nil)
	}

	// ensure user exists
	u, err := FindUserByID(userId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update user
	u.Update(user.Name, user.Description, user.Email, user.Role, user.Notifications)

	payload := u.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
