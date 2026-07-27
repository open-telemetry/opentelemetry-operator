// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus

import "slices"

// AvailableCRDs represents the list of monitoring.coreos.com CRD resource names
// available in the cluster (e.g. "servicemonitors", "podmonitors", "probes", "scrapeconfigs").
type AvailableCRDs []string

func (a AvailableCRDs) Available() bool {
	return len(a) > 0
}

func (a AvailableCRDs) AvailableServiceMonitor() bool {
	return slices.Contains(a, "servicemonitors")
}

func (a AvailableCRDs) AvailablePodMonitor() bool {
	return slices.Contains(a, "podmonitors")
}
