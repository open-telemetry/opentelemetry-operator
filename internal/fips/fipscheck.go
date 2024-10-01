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
	"strings"
)

type FIPSCheck interface {
	DisabledComponents(receivers map[string]interface{}, exporters map[string]interface{}, processors map[string]interface{}, extensions map[string]interface{}) []string
}

// FipsCheck holds configuration for FIPS black list.
type fipsCheck struct {
	isFIPSEnabled bool

	receivers  map[string]bool
	exporters  map[string]bool
	processors map[string]bool
	extensions map[string]bool
}

type noopFIPSCheck struct{}

func (noopFIPSCheck) DisabledComponents(receivers map[string]interface{}, exporters map[string]interface{}, processors map[string]interface{}, extensions map[string]interface{}) []string {
	return nil
}

// NewFipsCheck creates new FipsCheck.
// It checks if FIPS is enabled on the platform in /proc/sys/crypto/fips_enabled.
func NewFipsCheck(FIPSEnabled bool, receivers, exporters, processors, extensions []string) FIPSCheck {
	if !FIPSEnabled {
		return &noopFIPSCheck{}
	}

	return &fipsCheck{
		receivers:  listToMap(receivers),
		exporters:  listToMap(exporters),
		processors: listToMap(processors),
		extensions: listToMap(extensions),
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
func (fips fipsCheck) DisabledComponents(receivers map[string]interface{}, exporters map[string]interface{}, processors map[string]interface{}, extensions map[string]interface{}) []string {
	var disabled []string
	if comp := isDisabled(fips.receivers, receivers); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isDisabled(fips.exporters, exporters); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isDisabled(fips.processors, processors); comp != "" {
		disabled = append(disabled, comp)
	}
	if comp := isDisabled(fips.extensions, extensions); comp != "" {
		disabled = append(disabled, comp)
	}
	return disabled
}

func isDisabled(blackListed map[string]bool, cfg map[string]interface{}) string {
	for id := range cfg {
		component := strings.Split(id, "/")[0]
		if blackListed[component] {
			return component
		}
	}
	return ""
}
