/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"encoding/base64"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/sethvargo/go-password/password"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

const (
	templatePasswordSymbolsCount          = 0
	templatePasswordLowercaseOnly         = false
	templatePasswordAllowRepeatCharacters = false
)

var TemplateFunctions = map[string]any{
	"b64encode": func(str string) string {
		return base64.StdEncoding.EncodeToString([]byte(str))
	},
	"generatePassword": func(n int) string {
		return GenerateRandomPassword(n)
	},
	"lower": func(str string) string {
		return strings.ToLower(str)
	},
	"set": func(d map[string]interface{}, key string, value interface{}) map[string]interface{} {
		d[key] = value
		return d
	},
	"sha512": func(in string) string {
		return Shasum512(in)
	},
}

func GenerateRandomPassword(n int) string {
	res, err := password.Generate(n, n/3, templatePasswordSymbolsCount, templatePasswordLowercaseOnly, templatePasswordAllowRepeatCharacters)
	if err != nil {
		klog.Error(err)
		return ""
	}
	return res
}

func Shasum512(in string) string {
	c := crypt.New(crypt.SHA512)
	s := sha512_crypt.GetSalt()
	saltString := string(s.GenerateWRounds(s.SaltLenMax, 4096))
	shadowHash, err := c.Generate([]byte(in), []byte(saltString))
	if err != nil {
		klog.Error(err)
		return ""
	}
	return shadowHash
}

func LoadTemplateFunctions(tpl *template.Template) {
	// Load sprig fts
	tpl.Funcs(sprig.FuncMap())
	// Load custom functions that may override the ones in sprig
	for k, function := range TemplateFunctions {
		tpl.Funcs(template.FuncMap{
			k: function,
		},
		)
	}
}
