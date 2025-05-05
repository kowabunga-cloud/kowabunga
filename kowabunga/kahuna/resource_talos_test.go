/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"testing"
)

func TestDownload(t *testing.T) {
	err := downloadTalosctlBinary("v1.9.4")
	if err != nil {
		t.Errorf("__")
	}
}
