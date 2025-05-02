/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kdomanski/iso9660"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
	"gopkg.in/yaml.v3"
)

const (
	CloudInitIsoDevice = "/dev/sr0"
	CloudInitMetaData  = "meta-data"

	HttpHeaderKowabungaSourceIP   = "X-Kowabunga-Source-Ip"
	HttpHeaderKowabungaInstanceID = "X-Kowabunga-Instance-Id"

	ErrorCloudInitUnsupportImage = "Unsupported Cloud Init image"
	ErrorInvalidInstanceMetdata  = "Invalid instance metadata"
)

type CloudInitInstanceMetaDataSettings struct {
	InstanceID           string `yaml:"instance-id"`
	MetadataEndpoint     string `yaml:"kowabunga_metadata_uri"`
	LocalIP              string `yaml:"kowabunga_local_ip"`
	ControllerEndpoint   string `yaml:"kowabunga_controller_uri"`
	ControllerAgentID    string `yaml:"kowabunga_controller_agent_id"`
	ControllerAgentToken string `yaml:"kowabunga_controller_agent_token"`
}

func GetCloudInitMetadataDataSettings() (CloudInitInstanceMetaDataSettings, error) {

	settings := CloudInitInstanceMetaDataSettings{}

	klog.Infof("Reading CloudInit configuration ...")
	image, err := os.Open(CloudInitIsoDevice)
	if err != nil {
		return settings, err
	}
	defer func() {
		_ = image.Close()
	}()

	img, err := iso9660.OpenImage(image)
	if err != nil {
		return settings, err
	}

	root, err := img.RootDir()
	if err != nil {
		return settings, err
	}

	if !root.IsDir() {
		return settings, fmt.Errorf("%s", ErrorCloudInitUnsupportImage)
	}

	children, err := root.GetChildren()
	if err != nil {
		return settings, err
	}

	for _, c := range children {
		if c.Name() == CloudInitMetaData {
			buf := bytes.NewBuffer(nil)
			_, err := io.Copy(buf, c.Reader())
			if err != nil {
				return settings, err
			}

			err = yaml.Unmarshal(buf.Bytes(), &settings)
			if err != nil {
				return settings, err
			}
			break
		}
	}

	return settings, nil
}

func GetInstanceMetadataRaw(settings CloudInitInstanceMetaDataSettings) (map[string]any, error) {
	var instanceMetadata map[string]any

	klog.Infof("Fetching Instance Metadata ...")
	req, err := http.NewRequest("GET", settings.MetadataEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(HttpHeaderKowabungaSourceIP, settings.LocalIP)
	req.Header.Set(HttpHeaderKowabungaInstanceID, settings.InstanceID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", ErrorInvalidInstanceMetdata)
	}

	err = json.NewDecoder(resp.Body).Decode(&instanceMetadata)
	if err != nil {
		return nil, err
	}

	return instanceMetadata, nil
}

func MetadataRawToStruct(meta map[string]any) (*metadata.InstanceMetadata, error) {
	var instanceMetadata metadata.InstanceMetadata

	raw, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &instanceMetadata)
	if err != nil {
		return nil, err
	}

	return &instanceMetadata, nil
}

func GetInstanceMetadata(settings CloudInitInstanceMetaDataSettings) (*metadata.InstanceMetadata, error) {
	meta, err := GetInstanceMetadataRaw(settings)
	if err != nil {
		return nil, err
	}
	return MetadataRawToStruct(meta)
}
