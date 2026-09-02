// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fips

import (
	"strings"
)

// ParseFipsFlag parses a comma-separated list of FIPS-disabled components (e.g. "receiver.otlp,exporter.debug")
// and returns them grouped by component type.
func ParseFipsFlag(fipsFlag string) (receivers, exporters, processors, extensions []string) {
	split := strings.SplitSeq(fipsFlag, ",")
	for val := range split {
		val = strings.TrimSpace(val)
		typeAndName := strings.Split(val, ".")
		if len(typeAndName) == 2 {
			componentType := typeAndName[0]
			name := typeAndName[1]

			switch componentType {
			case "receiver":
				receivers = append(receivers, name)
			case "exporter":
				exporters = append(exporters, name)
			case "processor":
				processors = append(processors, name)
			case "extension":
				extensions = append(extensions, name)
			}
		}
	}
	return receivers, exporters, processors, extensions
}
