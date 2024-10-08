// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
