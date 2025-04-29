/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/agents"
)

type KaktusAgentConfig struct {
	Global  agents.KowabungaAgentGlobalConfig `yaml:"global"`
	Libvirt KaktusAgentLibvirtConfig          `yaml:"libvirt"`
	Ceph    KaktusAgentCephConfig             `yaml:"ceph"`
}

type KaktusAgentLibvirtConfig struct {
	Protocol string                      `yaml:"protocol"`
	Address  string                      `yaml:"address"`
	Port     int                         `yaml:"port"`
	TLS      KaktusAgentLibvirtTlsConfig `yaml:"tls,omitempty"`
}

type KaktusAgentLibvirtTlsConfig struct {
	PrivateKey string `yaml:"key"`
	PublicCert string `yaml:"cert"`
	CA         string `yaml:"ca"`
}

type KaktusAgentCephConfig struct {
	PluginLib string                       `yaml:"plugin"`
	Monitor   KaktusAgentCephMonitorConfig `yaml:"monitor"`
}

type KaktusAgentCephMonitorConfig struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

func KaktusConfigParser(f *os.File) (*KaktusAgentConfig, error) {
	var config KaktusAgentConfig

	// unmarshal configuration
	contents, _ := io.ReadAll(f)
	defer f.Close()
	err := yaml.Unmarshal(contents, &config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}
