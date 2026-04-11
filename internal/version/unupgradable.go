// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import "github.com/open-telemetry/opentelemetry-operator/pkg/constants"

// unupgradableCollectorVersions contains collector versions that cannot be automatically upgraded from.
// Key is the version string (e.g., "0.99.0") for O(1) lookup.
// Value is a custom warning message explaining why and how to upgrade manually.
var unupgradableCollectorVersions = map[string]string{
	// Example entry (commented out until needed):
	// "0.99.0": "Breaking changes in configuration format require manual migration. See https://github.com/open-telemetry/opentelemetry-operator/issues/XXXX",
}

// IsCollectorVersionUnupgradable checks if a collector version cannot be automatically upgraded from.
// Returns true and the warning message if unupgradable, false and empty string otherwise.
func IsCollectorVersionUnupgradable(version string) (notUpgradable bool, message string) {
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

// unupgradableInstrumentationVersions contains instrumentation versions that cannot be automatically upgraded from.
// Outer key is the language, inner key is the version string (image tag), value is the warning message.
var unupgradableInstrumentationVersions = map[constants.InstrumentationLanguage]map[string]string{
	// Used by e2e tests in tests/e2e-instrumentation/instrumentation-blocked-upgrade/.
	constants.InstrumentationLanguageJava: {
		"0.0.0-blocked-for-testing": "This version is blocked for e2e testing purposes.",
	},
}

// IsInstrumentationVersionUnupgradable checks if an instrumentation version for a specific language cannot be automatically upgraded from.
// The imageOrVersion parameter can be either a full image reference (e.g., "ghcr.io/org/image:v1.0.0") or just a version tag.
// Returns true and the warning message if unupgradable, false and empty string otherwise.
func IsInstrumentationVersionUnupgradable(language constants.InstrumentationLanguage, imageOrVersion string) (notUpgradable bool, message string) {
	langVersions, exists := unupgradableInstrumentationVersions[language]
	if !exists {
		return false, ""
	}

	// Extract version from image if it contains ":"
	version := extractVersionFromImage(imageOrVersion)

	if msg, exists := langVersions[version]; exists {
		return true, msg
	}
	return false, ""
}

// extractVersionFromImage extracts the tag/version from a container image reference.
// If the input doesn't contain ":", it's returned as-is (assumed to be just a version).
// Examples:
//   - "ghcr.io/org/image:v1.0.0" -> "v1.0.0"
//   - "v1.0.0" -> "v1.0.0"
//   - "ghcr.io/org/image" -> "latest" (no tag means latest)
func extractVersionFromImage(image string) string {
	if image == "" {
		return ""
	}

	// Find the last colon that's not part of a port in the registry
	lastColon := -1
	slashAfterColon := false
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			lastColon = i
			break
		}
		if image[i] == '/' {
			slashAfterColon = true
		}
	}

	// If there's a colon and no slash after it, it's a tag
	if lastColon != -1 && !slashAfterColon {
		return image[lastColon+1:]
	}

	// If no tag found and image contains "/", return "latest"
	for i := 0; i < len(image); i++ {
		if image[i] == '/' {
			return "latest"
		}
	}

	// No "/" means it might be just a version string
	return image
}

// SetUnupgradableInstrumentationVersionsForTests allows tests to set custom unupgradable instrumentation versions.
// Returns a cleanup function that restores the original map.
// This function should only be used in tests.
func SetUnupgradableInstrumentationVersionsForTests(versions map[constants.InstrumentationLanguage]map[string]string) func() {
	original := unupgradableInstrumentationVersions
	unupgradableInstrumentationVersions = versions
	return func() {
		unupgradableInstrumentationVersions = original
	}
}
