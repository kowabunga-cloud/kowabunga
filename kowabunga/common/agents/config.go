/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

type KowabungaAgentGlobalConfig struct {
	LogLevel string `yaml:"logLevel"`
	ID       string `yaml:"id"`
	Endpoint string `yaml:"endpoint"`
	APIKey   string `yaml:"apiKey"`
}
