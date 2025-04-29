/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TestCloudInitConfigDir = "/tmp/kowabunga/cloud-init"
)

func TestWindowsUserDataTemplate(t *testing.T) {
	testResultDir := TestCloudInitConfigDir
	err := os.MkdirAll(testResultDir, 0777)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	res := Resource{
		ID:   primitive.NewObjectID(),
		Name: "roottest",
	}
	sub := &Subnet{
		Resource: res,
		CIDR:     "10.0.0.0/24",
		Gateway:  "10.0.0.1",
		DNS:      "superdns",
		Reserved: []*IPRange{
			{
				First: "10.0.0.1",
				Last:  "10.0.0.5",
			},
		},
		GwPool: []*IPRange{
			{
				First: "10.0.0.250",
				Last:  "10.0.0.252",
			},
		},
		Routes: []string{
			"10.3.0.0/24",
		},
		Application: "test",
		AdapterIDs:  []string{},
	}

	routesByInterface := make(map[string]Subnet)
	routesByInterface["dummyadapter"] = *sub
	data := UserDataSettings{
		Hostname:          "test-host",
		Domain:            "superdomain.com",
		RootPassword:      "superpass",
		ServiceUser:       "kowabunga",
		ServiceUserPubKey: "randomKey",
		MetadataAlias:     "curl smthg",
		InterfacesSubnet:  routesByInterface,
	}

	ci := &CloudInit{
		Name:     "ci",
		OS:       "windows",
		TmpDir:   testResultDir,
		IsoImage: "windows",
		IsoSize:  10,
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	tpl := filepath.Join(dir, "/../../config/templates/windows/user_data.yml")
	err = ci.SetData(tpl, "kw_user_data_tests", data)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
}
