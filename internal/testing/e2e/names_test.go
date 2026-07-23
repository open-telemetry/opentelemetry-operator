// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestDNSNameFromT(t *testing.T) {
	name := DNSNameFromT(t)
	assert.Empty(t, validation.IsDNS1123Label(name))
	assert.True(t, strings.HasPrefix(name, "testdnsnamefromt-"), "name %q should be derived from the test name", name)
	assert.Len(t, name, len("testdnsnamefromt")+1+randomSuffixLen)
}

func TestNamespaceFromT(t *testing.T) {
	t.Run("with subtest, spaces And UPPERCASE plus a name long enough to overflow the sixty-three character namespace limit", func(t *testing.T) {
		ns := NamespaceFromT(t)
		assert.Empty(t, validation.IsDNS1123Label(ns))
		assert.True(t, strings.HasPrefix(ns, "testnamespacefromt-with-subtest-spaces-and-uppercase"), "namespace %q should be derived from the test name", ns)
		assert.Len(t, ns, validation.DNS1123LabelMaxLength)
	})
}

func TestRandomName(t *testing.T) {
	for _, tt := range []struct {
		testName string
		base     string
		maxLen   int
		prefix   string
		length   int
	}{
		{
			testName: "invalid characters collapse into single hyphens",
			base:     "Test/Foo__Bar",
			maxLen:   63,
			prefix:   "test-foo-bar-",
			length:   len("test-foo-bar") + 1 + randomSuffixLen,
		},
		{
			testName: "base with no valid characters falls back to test prefix",
			base:     "___",
			maxLen:   63,
			prefix:   "test-",
			length:   len("test") + 1 + randomSuffixLen,
		},
		{
			testName: "truncation drops a trailing hyphen",
			base:     "abc-" + strings.Repeat("x", 63), // cut lands right after the '-'
			maxLen:   4 + randomSuffixLen + 1,
			prefix:   "abc-",
			length:   len("abc") + 1 + randomSuffixLen,
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			name := randomName(tt.base, tt.maxLen)
			assert.Empty(t, validation.IsDNS1123Label(name))
			assert.True(t, strings.HasPrefix(name, tt.prefix), "name %q should start with %q", name, tt.prefix)
			assert.Len(t, name, tt.length)
			assert.LessOrEqual(t, len(name), tt.maxLen)
		})
	}
}
