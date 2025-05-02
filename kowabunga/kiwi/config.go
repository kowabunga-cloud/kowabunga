/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
)

type KiwiAgentConfig struct {
	Global   agents.KowabungaAgentGlobalConfig `yaml:"global"`
	PowerDNS KiwiAgentPowerDnsConfig           `yaml:"pdns"`
}

type KiwiAgentPowerDnsConfig struct {
	AS        KiwiAgentPowerDnsAuthoritativeConfig `yaml:"authoritative"`
	Recursors []KiwiAgentPowerDnsRecursorConfig    `yaml:"recursors"`
}

type KiwiAgentPowerDnsAuthoritativeConfig struct {
	Host    string `yaml:"host"`
	APIPort int    `yaml:"api_port"`
	APIKey  string `yaml:"api_key"`
}

type KiwiAgentPowerDnsRecursorConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	APIPort int    `yaml:"api_port"`
	APIKey  string `yaml:"api_key"`
}

func KiwiConfigParser(f *os.File) (*KiwiAgentConfig, error) {
	var config KiwiAgentConfig

	// unmarshal configuration
	contents, _ := io.ReadAll(f)
	defer func() {
		_ = f.Close()
	}()
	err := yaml.Unmarshal(contents, &config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}
