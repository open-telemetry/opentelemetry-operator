// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

// Kubernetes admission warnings are transmitted in HTTP headers and MUST NOT contain control
// characters like newlines, so these messages are kept on a single line.
const (
	dotNet130WarningMessage = ".NET instrumentation 1.3.0 contains breaking HTTP semantic convention changes. " +
		"See https://opentelemetry.io/blog/2023/http-conventions-declared-stable/ for a general description of the changes. " +
		"To upgrade, manually set the image on this Instrumentation resource to 1.3.0 or higher."
)

// unupgradableInstrumentationVersions contains instrumentation versions that cannot be automatically upgraded from.
// Outer key is the language, inner key is the version tag, value is the warning message.
var unupgradableInstrumentationVersions = map[constants.InstrumentationLanguage]map[string]string{
	constants.InstrumentationLanguageDotNet: {
		"1.3.0": dotNet130WarningMessage,
	},
}

// IsInstrumentationVersionUnupgradable checks if an instrumentation image upgrade should be blocked.
// It first verifies the image is from the same repository as defaultImage — images from other repositories are never blocked.
// Then it checks whether any blocked version falls in the range between the current image tag and the default image tag (inclusive).
// This catches both exact matches and upgrades that would skip over a blocked version.
// Returns true and the warning message if blocked, false and empty string otherwise.
func IsInstrumentationVersionUnupgradable(language constants.InstrumentationLanguage, image, defaultImage string) (blocked bool, warning string) {
	imageRepo, imageTag := splitImage(image)
	defaultRepo, defaultTag := splitImage(defaultImage)
	if imageRepo != defaultRepo {
		return false, ""
	}

	langVersions, exists := unupgradableInstrumentationVersions[language]
	if !exists {
		return false, ""
	}

	currentVer, err := semver.NewVersion(imageTag)
	if err != nil {
		// Can't parse current version, fall back to exact match
		if msg, exists := langVersions[imageTag]; exists {
			return true, msg
		}
		return false, ""
	}

	targetVer, err := semver.NewVersion(defaultTag)
	if err != nil {
		// Can't parse target version, fall back to exact match
		if msg, exists := langVersions[imageTag]; exists {
			return true, msg
		}
		return false, ""
	}

	for blockedTag, msg := range langVersions {
		blockedVer, err := semver.NewVersion(blockedTag)
		if err != nil {
			// Can't parse this blocked version, try exact match only
			if blockedTag == imageTag {
				return true, msg
			}
			continue
		}
		// Block if the blocked version is in range (current, target]
		if blockedVer.GreaterThan(currentVer) && blockedVer.LessThanEqual(targetVer) {
			return true, msg
		}
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
