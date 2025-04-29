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

func NewKomputeRouter() sdk.Router {
	return sdk.NewKomputeAPIController(&KomputeService{})
}

type KomputeService struct{}

func (s *KomputeService) DeleteKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	// ensure Kompute exists
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove Kompute
	err = k.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) ListKomputes(ctx context.Context) (sdk.ImplResponse, error) {
	komputes := FindKomputes()
	var payload []string
	for _, k := range komputes {
		payload = append(payload, k.String())
	}

	return HttpOK(payload)
}

func (s *KomputeService) ReadKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *KomputeService) ReadKomputeState(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload, err := k.GetState()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(payload)
}

func (s *KomputeService) RebootKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Reboot()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) ResetKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Reset()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) ResumeKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Resume()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) ShutdownKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Shutdown()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) StartKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Start()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) StopKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Stop()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) SuspendKompute(ctx context.Context, komputeId string) (sdk.ImplResponse, error) {
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = k.Suspend()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *KomputeService) UpdateKompute(ctx context.Context, komputeId string, kompute sdk.Kompute) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("komputeId", komputeId), RA("kompute", kompute))

	// check for params
	if kompute.Name == "" && kompute.Memory == 0 && kompute.Vcpus == 0 && kompute.Disk == 0 {
		return HttpBadParams(nil)
	}

	// ensure Kompute exists
	k, err := FindKomputeByID(komputeId)
	if err != nil {
		return HttpNotFound(err)
	}

	// find associated instance
	i, err := k.Instance()
	if err != nil {
		return HttpNotFound(err)
	}

	// find associated project
	prj, err := k.Project()
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure we're allowed by quotas
	cpuDelta := kompute.Vcpus - i.CPU
	memDelta := kompute.Memory - i.Memory
	if cpuDelta > 0 || memDelta > 0 {
		if !prj.AllowInstanceCreationOrUpdate(0, cpuDelta, memDelta) {
			return HttpQuota(nil)
		}
	}
	var size int64 = 0 // sum of all volume sizes
	for _, volumeId := range i.Volumes() {
		v, err := FindVolumeByID(volumeId)
		if err != nil {
			return HttpNotFound(err)
		}
		size += v.Size
	}
	sizeDelta := kompute.Disk - size
	if sizeDelta > 0 {
		if !prj.AllowVolumeCreationOrUpdate(sizeDelta) {
			return HttpQuota(nil)
		}
	}

	// update Kompute
	err = k.Update(kompute.Name, kompute.Description, kompute.Vcpus, kompute.Memory, kompute.Disk, kompute.DataDisk)
	if err != nil {
		return HttpServerError(err)
	}

	payload := k.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
