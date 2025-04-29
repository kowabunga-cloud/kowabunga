/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewVolumeRouter() sdk.Router {
	return sdk.NewVolumeAPIController(&VolumeService{})
}

type VolumeService struct{}

func (s *VolumeService) DeleteVolume(ctx context.Context, volumeId string) (sdk.ImplResponse, error) {
	// ensure volume exists
	v, err := FindVolumeByID(volumeId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove volume
	err = v.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *VolumeService) ListVolumes(ctx context.Context) (sdk.ImplResponse, error) {
	volumes := FindVolumes()
	var payload []string
	for _, v := range volumes {
		payload = append(payload, v.String())
	}

	return HttpOK(payload)
}

func (s *VolumeService) ReadVolume(ctx context.Context, volumeId string) (sdk.ImplResponse, error) {
	v, err := FindVolumeByID(volumeId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

// UpdateVolume -
func (s *VolumeService) UpdateVolume(ctx context.Context, volumeId string, volume sdk.Volume) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("volumeId", volumeId), RA("volume", volume))

	// check for params
	if volume.Name == "" && volume.Description == "" && volume.Size == 0 {
		return HttpBadParams(nil)
	}

	// ensure volume exists
	v, err := FindVolumeByID(volumeId)
	if err != nil {
		return HttpNotFound(err)
	}

	// find associated project
	prj, err := v.Project()
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure we're allowed by quotas
	sizeDelta := volume.Size - v.Size
	if sizeDelta > 0 {
		if !prj.AllowVolumeCreationOrUpdate(sizeDelta) {
			return HttpQuota(err)
		}
	}

	// update volume
	err = v.Update(volume.Name, volume.Description, volume.Size)
	if err != nil {
		return HttpServerError(err)
	}

	payload := v.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
