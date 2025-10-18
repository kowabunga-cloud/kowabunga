//go:build linux
// +build linux

/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"github.com/ceph/go-ceph/rados"
)

type ccs struct {
	// Settings
	Name    string
	Address string
	Port    int

	// Connection
	keepRunning bool
	Conn        *rados.Conn

	// CLI
	Bin string
}

// this is our plugin's exported symbol
var CephConnectionSettings ccs
