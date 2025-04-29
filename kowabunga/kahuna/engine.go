/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

type KahunaEngine struct {
	ApiRouters []sdk.Router
	Exporter   *KowabungaExporter
}

func (ke *KahunaEngine) PreFlight(cfg KowabungaConfig) error {
	// register global config
	SetCfg(&cfg)

	// database connection
	err := GetDB().Open(cfg.Global.DB.URI, cfg.Global.DB.Name)
	if err != nil {
		klog.Errorf("Unable to connect to MongoDB database: %s", err)
		return err
	}

	return nil
}

func (ke *KahunaEngine) Cleanup() {
	err := GetDB().Close()
	if err != nil {
		klog.Errorf("%v", err)
	}
}

func (ke *KahunaEngine) RegisterApiHandlers() {
	// register API handlers
	routers := []sdk.Router{
		NewAdapterRouter(),
		NewAgentRouter(),
		NewDnsRecordRouter(),
		NewInstanceRouter(),
		NewKaktusRouter(),
		NewKawaiiRouter(),
		NewKiwiRouter(),
		NewKomputeRouter(),
		NewKonveyRouter(),
		NewKyloRouter(),
		NewNfsRouter(),
		NewProjectRouter(),
		NewRegionRouter(),
		NewStoragePoolRouter(),
		NewSubnetRouter(),
		NewTeamRouter(),
		NewTemplateRouter(),
		NewTokenRouter(),
		NewUserRouter(),
		NewVNetRouter(),
		NewVolumeRouter(),
		NewZoneRouter(),
	}
	ke.ApiRouters = append(ke.ApiRouters, routers...)
}

func (ke *KahunaEngine) MigrateDatabase(cfg KowabungaConfig) error {
	// disable cache
	GetCache().Init(false, cfg.Global.Cache.Type, cfg.Global.Cache.Size, cfg.Global.Cache.TTL)

	return MigrateDatabaseSchema()
}

func (ke *KahunaEngine) Run(cfg KowabungaConfig) {

	defer ke.Cleanup()

	// cache initialization
	GetCache().Init(cfg.Global.Cache.Enabled, cfg.Global.Cache.Type, cfg.Global.Cache.Size, cfg.Global.Cache.TTL)

	// register prometheus exporter
	ke.Exporter = NewExporter()

	// register API handlers
	ke.RegisterApiHandlers()

	srv := NewHTTPServer(ke, cfg.Global.HTTP.Address, cfg.Global.HTTP.Port)
	defer func() {
		if err := srv.Shutdown(); err != nil {
			// error handle
			klog.Error(err)
		}
	}()

	if err := srv.Serve(); err != nil {
		klog.Error(err)
	}
}
