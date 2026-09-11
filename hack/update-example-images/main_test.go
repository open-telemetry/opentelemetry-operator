// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"maps"
	"slices"
	"testing"
)

var testRefs = map[string]string{
	"java":        "ghcr.io/otel/autoinstrumentation-java:2.0.0-1",
	"nodejs":      "ghcr.io/otel/autoinstrumentation-nodejs:0.1.0-1",
	"python":      "ghcr.io/otel/autoinstrumentation-python:0.2b0-1",
	"dotnet":      "ghcr.io/otel/autoinstrumentation-dotnet:1.0.0-1",
	"go":          "ghcr.io/otel/autoinstrumentation-go:v0.3.0",
	"apacheHttpd": "ghcr.io/otel/autoinstrumentation-apache-httpd:1.0.0-1",
	"nginx":       "ghcr.io/otel/autoinstrumentation-apache-httpd:1.0.0-1",
}

func TestPinImages(t *testing.T) {
	headerRe := headerRegex(slices.Sorted(maps.Keys(testRefs)))

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "replaces existing first-child image",
			in: "spec:\n" +
				"  java:\n" +
				"    image: old:tag\n" +
				"    env: []\n",
			want: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env: []\n",
		},
		{
			name: "replaces first-child image and keeps later siblings",
			in: "spec:\n" +
				"  apacheHttpd:\n" +
				"    image: old:tag # comment\n" +
				"    version: \"2.2\"\n",
			want: "spec:\n" +
				"  apacheHttpd:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-apache-httpd:1.0.0-1\n" +
				"    version: \"2.2\"\n",
		},
		{
			name: "replaces first-child image in multiple language blocks",
			in: "spec:\n" +
				"  java:\n" +
				"    image: old-java\n" +
				"    env: []\n" +
				"  exporter:\n" +
				"    endpoint: http://x:4317\n" +
				"  python:\n" +
				"    image: old-python\n",
			want: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env: []\n" +
				"  exporter:\n" +
				"    endpoint: http://x:4317\n" +
				"  python:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-python:0.2b0-1\n",
		},
		{
			name: "leaves a language block with no image untouched (never inserts)",
			in: "spec:\n" +
				"  nodejs:\n" +
				"    env:\n" +
				"      - name: NODE_PATH\n" +
				"        value: /app\n",
			want: "spec:\n" +
				"  nodejs:\n" +
				"    env:\n" +
				"      - name: NODE_PATH\n" +
				"        value: /app\n",
		},
		{
			name: "does not touch an image that is not the first child",
			in: "  apacheHttpd:\n" +
				"    version: \"2.2\"\n" +
				"    image: old:tag\n",
			want: "  apacheHttpd:\n" +
				"    version: \"2.2\"\n" +
				"    image: old:tag\n",
		},
		{
			name: "replaces first-child image but not a deeper image line",
			in: "  java:\n" +
				"    image: old\n" +
				"    volumeClaimTemplate:\n" +
				"      spec:\n" +
				"        image: not-a-lang-image\n",
			want: "  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    volumeClaimTemplate:\n" +
				"      spec:\n" +
				"        image: not-a-lang-image\n",
		},
		{
			name: "no trailing newline is preserved",
			in:   "  java:\n    image: old",
			want: "  java:\n    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1",
		},
		{
			name: "sdk-only content is unchanged",
			in:   "spec:\n  exporter:\n    endpoint: http://x:4317\n",
			want: "spec:\n  exporter:\n    endpoint: http://x:4317\n",
		},
		{
			name: "is idempotent on already-pinned content",
			in: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env: []\n",
			want: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env: []\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pinImages(tt.in, headerRe, testRefs)
			if got != tt.want {
				t.Errorf("pinImages() mismatch\n--- got ---\n%q\n--- want ---\n%q", got, tt.want)
			}
			if again := pinImages(got, headerRe, testRefs); again != got {
				t.Errorf("pinImages() not idempotent\n--- first ---\n%q\n--- second ---\n%q", got, again)
			}
		})
	}
}

func TestLanguageKeys(t *testing.T) {
	got, err := languageKeys()
	if err != nil {
		t.Fatalf("languageKeys() error: %v", err)
	}
	want := []string{"apacheHttpd", "dotnet", "go", "java", "nginx", "nodejs", "python"}
	if !slices.Equal(got, want) {
		t.Errorf("languageKeys() = %v, want %v", got, want)
	}
}
