// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

// unupgradableInstrumentationVersions contains instrumentation versions that cannot be automatically upgraded from.
// Outer key is the language, inner key is the version tag, value is the warning message.
var unupgradableInstrumentationVersions = map[constants.InstrumentationLanguage]map[string]string{
	// Used by e2e tests in tests/e2e-instrumentation/instrumentation-blocked-upgrade/.
	constants.InstrumentationLanguageJava: {
		"0.0.0-blocked-for-testing": "This version is blocked for e2e testing purposes.",
	},
}

// IsInstrumentationVersionUnupgradable checks if an instrumentation image for a specific language cannot be automatically upgraded from.
// It first verifies the image is from the same repository as defaultImage — images from other repositories are never blocked.
// Returns true and the warning message if unupgradable, false and empty string otherwise.
func IsInstrumentationVersionUnupgradable(language constants.InstrumentationLanguage, image, defaultImage string) (notUpgradable bool, warning string) {
	imageRepo, imageTag := splitImage(image)
	defaultRepo, _ := splitImage(defaultImage)
	if imageRepo != defaultRepo {
		return false, ""
	}

	langVersions, exists := unupgradableInstrumentationVersions[language]
	if !exists {
		return false, ""
	}
	if msg, exists := langVersions[imageTag]; exists {
		return true, msg
	}
	return false, ""
}

// splitImage splits a container image reference into repository and tag.
// For "ghcr.io/org/image:v1.0.0" it returns ("ghcr.io/org/image", "v1.0.0").
// For "ghcr.io/org/image" (no tag) it returns ("ghcr.io/org/image", "latest").
func splitImage(image string) (repo, tag string) {
	// Find the last colon; if there's a slash after it, it's a port separator, not a tag separator.
	lastColon := strings.LastIndex(image, ":")
	if lastColon != -1 && !strings.Contains(image[lastColon:], "/") {
		return image[:lastColon], image[lastColon+1:]
	}
	return image, "latest"
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
