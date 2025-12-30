/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"os"

	"github.com/kowabunga-cloud/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kahuna"
)

func main() {
	// parsing commands
	cfgFile, debug, migrate := kahuna.ParseCommands()

	cfg, err := kahuna.ParseConfig(cfgFile)
	if err != nil {
		fmt.Printf("config: unable to unmarshal config (%s)\n", err)
		os.Exit(1)
	}

	// init our logger
	logLevel := cfg.Global.LogLevel
	if debug {
		logLevel = "DEBUG"
	}
	klog.Init("kahuna", []klog.LoggerConfiguration{
		{
			Type:    "console",
			Enabled: true,
			Level:   logLevel,
		},
	})

	// register everything
	var ae = &kahuna.KahunaEngine{}
	err = ae.PreFlight(cfg)
	if err != nil {
		klog.Errorf("Unable to start: %s", err)
		os.Exit(1)
	}

	if migrate {
		err = ae.MigrateDatabase(cfg)
		if err != nil {
			klog.Error(err)
			os.Exit(1)
		}
	} else {
		ae.Run(cfg)
	}

	os.Exit(0)
}
