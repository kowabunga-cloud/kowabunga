/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package common

const (
	KowabungaKaktusAgent     = "Kaktus"
	KowabungaKiwiAgent       = "Kiwi"
	KowabungaControllerAgent = "Kontroller"
)

func SupportedAgents() []string {
	return []string{
		KowabungaKaktusAgent,
		KowabungaKiwiAgent,
		KowabungaControllerAgent,
	}
}
