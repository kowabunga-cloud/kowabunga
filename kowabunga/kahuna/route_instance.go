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

func NewInstanceRouter() sdk.Router {
	return sdk.NewInstanceAPIController(&InstanceService{})
}

type InstanceService struct{}

func (s *InstanceService) DeleteInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	// ensure instance exists
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure there's no children referenced
	if i.HasChildren() {
		return HttpConflict(nil)
	}

	// remove instance
	err = i.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) ListInstances(ctx context.Context) (sdk.ImplResponse, error) {
	instances := FindInstances()
	var payload []string
	for _, i := range instances {
		payload = append(payload, i.String())
	}

	return HttpOK(payload)
}

func (s *InstanceService) ReadInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := i.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *InstanceService) ReadInstanceRemoteConnection(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload, err := i.GetRemoteConnectionURL()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(payload)
}

func (s *InstanceService) ReadInstanceState(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload, err := i.GetState()
	if err != nil {
		return HttpNotFound(err)
	}

	return HttpOK(payload)
}

func (s *InstanceService) RebootInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = i.Reboot()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) ResetInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = i.Reset()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) ResumeInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	vm, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = vm.Resume()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) ShutdownInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = i.Shutdown()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) StartInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = i.Start()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) StopInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = i.Stop()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) SuspendInstance(ctx context.Context, instanceId string) (sdk.ImplResponse, error) {
	vm, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	err = vm.Suspend()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *InstanceService) UpdateInstance(ctx context.Context, instanceId string, instance sdk.Instance) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("instanceId", instanceId), RA("instance", instance))

	// check for params
	if instance.Name == "" && instance.Memory == 0 && instance.Vcpus == 0 {
		return HttpBadParams(nil)
	}

	// ensure instance exists
	i, err := FindInstanceByID(instanceId)
	if err != nil {
		return HttpNotFound(err)
	}

	// find associated project
	prj, err := i.Project()
	if err != nil {
		return HttpNotFound(err)
	}

	// ensure we're allowed by quotas
	cpuDelta := instance.Vcpus - i.CPU
	memDelta := instance.Memory - i.Memory
	if cpuDelta > 0 || memDelta > 0 {
		if !prj.AllowInstanceCreationOrUpdate(0, cpuDelta, memDelta) {
			return HttpQuota(nil)
		}
	}

	// update instance
	err = i.Update(instance.Name, instance.Description, instance.Vcpus, instance.Memory, instance.Adapters, instance.Volumes)
	if err != nil {
		return HttpServerError(err)
	}

	payload := i.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
