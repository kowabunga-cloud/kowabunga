/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sethvargo/go-password/password"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

const (
	MongoCollectionUserSchemaVersion = 2
	MongoCollectionUserName          = "user"

	ErrUserNoSuchToken = "no such token in user"

	UserRoleSuperAdmin   = "superAdmin"
	UserRoleProjectAdmin = "projectAdmin"
	UserRoleStandard     = "user"

	UserRegistrationTokenLength                = 16
	UserRegistrationTokenDigitsCount           = 4
	UserRegistrationTokenSymbolsCount          = 0
	UserRegistrationTokenLowercaseOnly         = true
	UserRegistrationTokenAllowRepeatCharacters = false

	UserPasswordLength                = 16
	UserPasswordDigitsCount           = 5
	UserPasswordSymbolsCount          = 0
	UserPasswordLowercaseOnly         = false
	UserPasswordAllowRepeatCharacters = true
	UserPasswordMinEntropyBits        = 70
	UserPasswordHashCost              = 10
)

type User struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents

	// properties
	Email                string `bson:"email"`
	PasswordHash         string `bson:"password_hash"`
	Role                 string `bson:"role"`
	PasswordRenewalToken string `bson:"password_renewal_token"`
	RegistrationToken    string `bson:"registration_token"`
	Enabled              bool   `bson:"enabled"`
	NotificationsEnabled bool   `bson:"notifications_enabled"`
	TokenID              string `bson:"token"`
	JWT                  string `bson:"jwt"` // ephemeral JWT authentication token

	// children references
	TeamIDs []string `bson:"team_ids"`
}

func UserMigrateSchema() error {
	// rename collection
	err := GetDB().RenameCollection("users", MongoCollectionUserName)
	if err != nil {
		return err
	}

	for _, user := range FindUsers() {
		if user.SchemaVersion == 0 || user.SchemaVersion == 1 {
			err := user.migrateSchemaV2()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewUser(name, desc, email, role string, notifications bool) (*User, error) {
	// verify role
	switch role {
	case UserRoleSuperAdmin, UserRoleProjectAdmin, UserRoleStandard:
		break
	default:
		role = UserRoleStandard
	}

	u := User{
		Resource:             NewResource(name, desc, MongoCollectionUserSchemaVersion),
		Role:                 role,
		NotificationsEnabled: notifications,
		TokenID:              "",
		JWT:                  "",
		TeamIDs:              []string{},
	}

	// verify email
	err := VerifyEmail(email)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	u.Email = email

	// generate a user registration token
	registrationToken, err := password.Generate(UserRegistrationTokenLength, UserRegistrationTokenDigitsCount,
		UserRegistrationTokenSymbolsCount, UserRegistrationTokenLowercaseOnly,
		UserRegistrationTokenAllowRepeatCharacters)
	if err != nil {
		return nil, fmt.Errorf("unable to generate new user registration token: %v", err)
	}
	u.RegistrationToken = registrationToken

	// notify user of account creation
	err = NewEmailUserCreated(&u)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	_, err = GetDB().Insert(MongoCollectionUserName, u)
	if err != nil {
		return nil, err
	}

	klog.Debugf("Created new user %s (%s)", u.String(), u.Name)

	return &u, nil
}

func FindUsers() []User {
	return FindResources[User](MongoCollectionUserName)
}

func FindUserByID(id string) (*User, error) {
	return FindResourceByID[User](MongoCollectionUserName, id)
}

func FindUserByName(name string) (*User, error) {
	return FindResourceByName[User](MongoCollectionUserName, name)
}

func FindUserByEmail(email string) (*User, error) {
	return FindResourceByEmail[User](MongoCollectionUserName, email)
}

func (u *User) renameDbField(from, to string) error {
	return GetDB().Rename(MongoCollectionUserName, u.ID, from, to)
}

func (u *User) setSchemaVersion(version int) error {
	return GetDB().SetSchemaVersion(MongoCollectionUserName, u.ID, version)
}

func (u *User) migrateSchemaV2() error {
	err := u.renameDbField("groups", "team_ids")
	if err != nil {
		return err
	}

	err = u.setSchemaVersion(2)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) Teams() []string {
	return u.TeamIDs
}

func (u *User) IsSuperAdmin() bool {
	return u.Role == UserRoleSuperAdmin
}

func (u *User) IsProjectAdmin() bool {
	return (u.Role == UserRoleProjectAdmin) || (u.Role == UserRoleSuperAdmin)
}

func (u *User) Token() (*Token, error) {
	return FindTokenByID(u.TokenID)
}

func (u *User) verifyPassword(password string) error {
	err := passwordvalidator.Validate(password, UserPasswordMinEntropyBits)
	if err != nil {
		klog.Error(err)
	}
	return err
}

func (u *User) hashPassword(password string) (string, error) {
	// verify password's strength
	err := u.verifyPassword(password)
	if err != nil {
		return "", err
	}

	// generate "hash" for DB storage
	hash, err := bcrypt.GenerateFromPassword([]byte(password), UserPasswordHashCost)
	if err != nil {
		return "", fmt.Errorf("unable to generate hash from provided password: %v", err)
	}

	return string(hash), nil
}

func (u *User) Enable() {
	u.RegistrationToken = ""
	u.Enabled = true
	u.Save()
}

func (u *User) JwtSession() (string, error) {
	signer := []byte(GetCfg().Global.JWT.Signature)

	// check for existing pre-signed valid token
	if u.JWT != "" {
		tokenString, _, err := VerifyJwt(u.JWT)
		if err != nil {
			klog.Debugf("Currently stored JWT for user %s is invalid or expired, will generate a new one", u.String())
		} else {
			klog.Debugf("Re-using previously generated JWT for user %s", u.String())
			return tokenString, nil
		}
	}

	// generate a new HMAC-signed token
	duration := time.Hour * time.Duration(GetCfg().Global.JWT.Lifetime)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			// see RFC7519 (https://datatracker.ietf.org/doc/html/rfc7519)
			"iss":   "Kowabunga",
			"uid":   u.String(),
			"email": u.Email,
			"role":  u.Role,
			"exp":   time.Now().Add(duration).Unix(),
		})
	tokenString, err := token.SignedString(signer)
	if err != nil {
		return "", err
	}

	// save up generated token
	u.JWT = tokenString
	u.Save()

	return tokenString, nil
}

func VerifyJwt(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		signer := []byte(GetCfg().Global.JWT.Signature)
		return signer, nil
	})
	if err != nil {
		klog.Error(err)
		return "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", err
	}

	return token.Raw, claims["uid"].(string), nil
}

func (u *User) Verify(password string) error {
	// comparing the supplied password with the hashed one from database
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

func (u *User) UpdatePassword(password string) error {
	// hash password for storage
	hp, err := u.hashPassword(password)
	if err != nil {
		klog.Error(err)
		return err
	}

	u.PasswordHash = hp
	u.Save()
	return nil
}

func (u *User) ResetPasswordRequest() error {
	// generate a user password renewal token
	passwordRenewalToken, err := password.Generate(UserRegistrationTokenLength, UserRegistrationTokenDigitsCount,
		UserRegistrationTokenSymbolsCount, UserRegistrationTokenLowercaseOnly,
		UserRegistrationTokenAllowRepeatCharacters)
	if err != nil {
		return fmt.Errorf("unable to generate new user password renewal token: %v", err)
	}
	u.PasswordRenewalToken = passwordRenewalToken
	u.Save()

	return NewEmailUserPasswordConfirmation(u)
}

func (u *User) ResetPassword() error {
	// invalidate password renewal token (one-time thing)
	u.PasswordRenewalToken = ""

	// generate a new robust user password
	userPassword, err := password.Generate(UserPasswordLength, UserPasswordDigitsCount,
		UserPasswordSymbolsCount, UserPasswordLowercaseOnly, UserPasswordAllowRepeatCharacters)
	if err != nil {
		return fmt.Errorf("unable to generate new robust user password: %v", err)
	}

	// send newly generated password by email
	err = NewEmailUserPassword(u, userPassword)
	if err != nil {
		return err
	}

	// hash password for storage
	hp, err := u.hashPassword(userPassword)
	if err != nil {
		klog.Error(err)
		return err
	}

	u.PasswordHash = hp
	u.Save()

	return nil
}

func (u *User) Update(name, desc, email, role string, notifications bool) {
	u.UpdateResourceDefaults(name, desc)

	if email != "" {
		// verify email
		err := VerifyEmail(email)
		if err != nil {
			klog.Error(err)
		}
		u.Email = email
	}

	if role != "" {
		u.Role = role
	}

	u.NotificationsEnabled = notifications
	u.Save()
}

func (u *User) Save() {
	u.Updated()
	_, err := GetDB().Update(MongoCollectionUserName, u.ID, u)
	if err != nil {
		klog.Error(err)
	}
}

func (u *User) Delete() error {
	klog.Debugf("Deleting user %s", u.String())

	if u.String() == ResourceUnknown {
		return nil
	}

	t, _ := u.Token()
	if t != nil {
		err := t.Delete()
		if err != nil {
			return err
		}
	}

	return GetDB().Delete(MongoCollectionUserName, u.ID)
}

func (u *User) Model() sdk.User {
	return sdk.User{
		Id:            u.String(),
		Name:          u.Name,
		Description:   u.Description,
		Email:         u.Email,
		Role:          u.Role,
		Notifications: u.NotificationsEnabled,
	}
}

// Tokens

func (u *User) AddToken(id string) {
	klog.Debugf("Adding token %s to agent %s", id, u.String())
	u.TokenID = id
	u.Save()
}

func (u *User) RemoveToken(id string) {
	klog.Debugf("Removing token %s from agent %s", id, u.String())
	u.TokenID = ""
	u.Save()
}

// Teams
func (u *User) AddTeam(id string) {
	klog.Debugf("Adding team %s to user %s", id, u.String())
	AddChildRef(&u.TeamIDs, id)
	u.Save() // save DB before looking back
}

func (u *User) RemoveTeam(id string) {
	klog.Debugf("Removing team %s from user %s", id, u.String())
	RemoveChildRef(&u.TeamIDs, id)
	u.Save()
}
