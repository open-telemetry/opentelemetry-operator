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
	"fmt"
	"os"
	"strings"
)

const fipsFile = "/proc/sys/crypto/fips_enabled"

// FipsCheck holds configuration for FIPS black list.
type FipsCheck struct {
	isFIPSEnabled bool

	receivers  map[string]bool
	exporters  map[string]bool
	processors map[string]bool
	extensions map[string]bool
}

// NewFipsCheck creates new FipsCheck.
// It checks if FIPS is enabled on the platform in /proc/sys/crypto/fips_enabled.
func NewFipsCheck(receivers, exporters, processors, extensions []string) FipsCheck {
	return FipsCheck{
		isFIPSEnabled: isFipsEnabled(),
		receivers:     listToMap(receivers),
		exporters:     listToMap(exporters),
		processors:    listToMap(processors),
		extensions:    listToMap(extensions),
	}
}

func listToMap(list []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range list {
		m[v] = true
	}
	return m
}

// Check checks if a submitted components are back lister or not.
func (fips FipsCheck) Check(receivers map[string]interface{}, exporters map[string]interface{}, processors map[string]interface{}, extensions map[string]interface{}) []string {
	if !fips.isFIPSEnabled {
		return nil
	}
	var disabled []string
	if comp := isBlackListed(fips.receivers, receivers); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isBlackListed(fips.exporters, exporters); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isBlackListed(fips.processors, processors); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isBlackListed(fips.extensions, extensions); comp != "" {
		disabled = append(disabled, comp)
	}
	return disabled
}

func isBlackListed(blackListed map[string]bool, cfg map[string]interface{}) string {
	for id := range cfg {
		component := strings.Split(id, "/")[0]
		if blackListed[component] {
			return component
		}
	}
	return ""
}

func isFipsEnabled() bool {
	// check if file exists
	if _, err := os.Stat(fipsFile); errors.Is(err, os.ErrNotExist) {
		fmt.Println("fips file doesn't exist")
		return false
	}
	content, err := os.ReadFile(fipsFile)
	if err != nil {
		// file cannot be read, enable FIPS to avoid any violations
		fmt.Println("cannot read fips file")
		return true
	}
	contentStr := string(content)
	contentStr = strings.TrimSpace(contentStr)
	return contentStr == "1"
}
