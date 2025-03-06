// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fips

import (
	"strings"
)

type FIPSCheck interface {
	// DisabledComponents checks if a submitted components are denied or not.
	DisabledComponents(receivers map[string]interface{}, exporters map[string]interface{}, processors map[string]interface{}, extensions map[string]interface{}) []string
}

// FipsCheck holds configuration for FIPS deny list.
type fipsCheck struct {
	receivers  map[string]bool
	exporters  map[string]bool
	processors map[string]bool
	extensions map[string]bool
}

// NewFipsCheck creates new FipsCheck.
func NewFipsCheck(receivers, exporters, processors, extensions []string) FIPSCheck {
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

func isDisabled(denyList map[string]bool, cfg map[string]interface{}) string {
	for id := range cfg {
		component := strings.Split(id, "/")[0]
		if denyList[component] {
			return component
		}
	}
	return ""
}
