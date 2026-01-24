// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

// unupgradableCollectorVersions contains collector versions that cannot be automatically upgraded from.
// Key is the version string (e.g., "0.99.0") for O(1) lookup.
// Value is a custom warning message explaining why and how to upgrade manually.
var unupgradableCollectorVersions = map[string]string{
	// Example entry (commented out until needed):
	// "0.99.0": "Breaking changes in configuration format require manual migration. See https://github.com/open-telemetry/opentelemetry-operator/issues/XXXX",
}

// IsCollectorVersionUnupgradable checks if a collector version cannot be automatically upgraded from.
// Returns true and the warning message if unupgradable, false and empty string otherwise.
func IsCollectorVersionUnupgradable(version string) (bool, string) {
	if msg, exists := unupgradableCollectorVersions[version]; exists {
		return true, msg
	}
	return false, ""
}

// SetUnupgradableCollectorVersionsForTests allows tests to set custom unupgradable versions.
// Returns a cleanup function that restores the original map.
// This function should only be used in tests.
func SetUnupgradableCollectorVersionsForTests(versions map[string]string) func() {
	original := unupgradableCollectorVersions
	unupgradableCollectorVersions = versions
	return func() {
		unupgradableCollectorVersions = original
	}
}
