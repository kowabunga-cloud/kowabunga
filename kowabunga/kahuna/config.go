/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"
	"io"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// config singleton
var cfgLock = &sync.Mutex{}
var kCFG *KowabungaConfig

func GetCfg() *KowabungaConfig {
	if kCFG == nil {
		cfgLock.Lock()
		defer cfgLock.Unlock()
	}

	return kCFG
}

func SetCfg(cfg *KowabungaConfig) {
	cfgLock.Lock()
	defer cfgLock.Unlock()
	kCFG = cfg
}

type KowabungaConfig struct {
	Global    KowabungaGlobalConfig    `yaml:"global"`
	CloudInit KowabungaCloudInitConfig `yaml:"cloudinit"`
}

type KowabungaGlobalConfig struct {
	LogLevel   string                   `yaml:"logLevel"`
	PublicURL  string                   `yaml:"publicUrl"`
	AdminEmail string                   `yaml:"adminEmail"`
	JWT        KowabungaJwtConfig       `yaml:"jwt"`
	HTTP       KowabungaHTTPConfig      `yaml:"http"`
	APIKey     string                   `yaml:"apiKey"`
	DB         KowabungaDBConfig        `yaml:"db"`
	Cache      KowabungaCacheConfig     `yaml:"cache"`
	Bootstrap  KowabungaBootstrapConfig `yaml:"bootstrap"`
	SMTP       KowabungaSmtpConfig      `yaml:"smtp"`
}

type KowabungaJwtConfig struct {
	Signature string `yaml:"signature"`
	Lifetime  int    `yaml:"lifetimeHours"`
}

type KowabungaHTTPConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type KowabungaDBConfig struct {
	URI  string `yaml:"uri"`
	Name string `yaml:"name"`
}

type KowabungaCacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
	Size    int    `yaml:"sizeMB"`
	TTL     int    `yaml:"expirationMinutes"`
}

type KowabungaBootstrapConfig struct {
	User   string `yaml:"user"`
	Pubkey string `yaml:"pubkey"`
}

type KowabungaSmtpConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type KowabungaCloudInitConfig struct {
	Linux   KowabungaCloudInitBaseConfig `yaml:"linux"`
	Windows KowabungaCloudInitBaseConfig `yaml:"windows"`
}

type KowabungaCloudInitBaseConfig struct {
	UserData      string `yaml:"userData"`
	MetaData      string `yaml:"metaData"`
	NetworkConfig string `yaml:"networkConfig"`
}

func ParseConfig(f *os.File) (KowabungaConfig, error) {
	var config KowabungaConfig

	// unmarshal configuration
	contents, _ := io.ReadAll(f)
	defer func() {
		_ = f.Close()
	}()
	err := yaml.Unmarshal(contents, &config)
	if err != nil {
		return config, err
	}

	// ensure default bootstrap settings are filled in
	if config.Global.Bootstrap.User == "" || config.Global.Bootstrap.Pubkey == "" {
		return config, fmt.Errorf("missing config bootstrap parameters")
	}

	// ensure cloud-init template files exist
	templates := []string{
		config.CloudInit.Linux.UserData,
		config.CloudInit.Linux.MetaData,
		config.CloudInit.Linux.NetworkConfig,
		config.CloudInit.Windows.UserData,
		config.CloudInit.Windows.MetaData,
		config.CloudInit.Windows.NetworkConfig,
	}
	for _, t := range templates {
		_, err = os.Stat(t)
		if err != nil {
			return config, err
		}
	}

	return config, nil
}
