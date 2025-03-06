// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fips

import (
	"errors"
	"os"
	"strings"
)

const fipsFile = "/proc/sys/crypto/fips_enabled"

// IsFipsEnabled checks whether FIPS is enabled on the platform.
func IsFipsEnabled() bool {
	// check if file exists
	if _, err := os.Stat(fipsFile); errors.Is(err, os.ErrNotExist) {
		return false
	}
	content, err := os.ReadFile(fipsFile)
	if err != nil {
		// file cannot be read, enable FIPS to avoid any violations
		return true
	}
	contentStr := string(content)
	contentStr = strings.TrimSpace(contentStr)
	return contentStr == "1"
}
