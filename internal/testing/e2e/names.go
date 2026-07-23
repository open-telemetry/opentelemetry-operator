// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"regexp"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// invalidNameChars matches runs of characters that may appear in a Go test name but
// not in a DNS-1123 name (e.g. '/' from subtests, '_' from spaces).
var invalidNameChars = regexp.MustCompile(`[^a-z0-9]+`)

// randomSuffixLen is the number of random hex characters appended to generated names
// so that repeated runs of the same test don't collide (e.g. while a previous run's
// namespace is still terminating).
const randomSuffixLen = 8

// DNSNameFromT generates a valid Kubernetes object name derived from the running
// test's name, with a random suffix for uniqueness across runs. The result is a
// DNS-1123 label (max 63 characters) — the strictest name format Kubernetes uses —
// so it is valid for any object, including Namespaces and Services.
func DNSNameFromT(t *testing.T) string {
	t.Helper()
	return randomName(t.Name(), validation.DNS1123LabelMaxLength)
}

// NamespaceFromT generates a valid Namespace name derived from the running test's
// name. It exists so namespace-generation call sites read clearly; the format is the
// same as DNSNameFromT's.
func NamespaceFromT(t *testing.T) string {
	t.Helper()
	return DNSNameFromT(t)
}

// randomName lowercases base, collapses each run of invalid characters into a single
// '-', truncates to leave room for the random suffix within maxLen, and appends the
// suffix.
func randomName(base string, maxLen int) string {
	prefix := strings.Trim(invalidNameChars.ReplaceAllString(strings.ToLower(base), "-"), "-")
	if maxPrefix := maxLen - randomSuffixLen - 1; len(prefix) > maxPrefix {
		prefix = strings.TrimRight(prefix[:maxPrefix], "-")
	}
	if prefix == "" {
		prefix = "test"
	}
	return envconf.RandomName(prefix, len(prefix)+1+randomSuffixLen)
}
