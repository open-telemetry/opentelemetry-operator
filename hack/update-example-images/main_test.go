// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

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
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "inserts image as first child",
			in: "spec:\n" +
				"  java:\n" +
				"    env:\n" +
				"    - name: X\n",
			want: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env:\n" +
				"    - name: X\n",
		},
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
			name: "moves a non-first-child image to first and keeps siblings",
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
			name: "handles multiple language blocks and ends block at less-indented sibling",
			in: "spec:\n" +
				"  java:\n" +
				"    env: []\n" +
				"  exporter:\n" +
				"    endpoint: http://x:4317\n" +
				"  python:\n" +
				"    env: []\n",
			want: "spec:\n" +
				"  java:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n" +
				"    env: []\n" +
				"  exporter:\n" +
				"    endpoint: http://x:4317\n" +
				"  python:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-python:0.2b0-1\n" +
				"    env: []\n",
		},
		{
			name: "preserves comments and blank lines inside a block",
			in: "  go:\n" +
				"    env:\n" +
				"      # keep me\n" +
				"\n" +
				"      - name: X\n",
			want: "  go:\n" +
				"    image: ghcr.io/otel/autoinstrumentation-go:v0.3.0\n" +
				"    env:\n" +
				"      # keep me\n" +
				"\n" +
				"      - name: X\n",
		},
		{
			name: "does not drop a deeper (non-direct-child) image line",
			in: "  java:\n" +
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
			in:   "  java:\n    env: []",
			want: "  java:\n    image: ghcr.io/otel/autoinstrumentation-java:2.0.0-1\n    env: []",
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
			got := pinImages(tt.in, testRefs)
			if got != tt.want {
				t.Errorf("pinImages() mismatch\n--- got ---\n%q\n--- want ---\n%q", got, tt.want)
			}
			// Running again must be a no-op.
			if again := pinImages(got, testRefs); again != got {
				t.Errorf("pinImages() not idempotent\n--- first ---\n%q\n--- second ---\n%q", got, again)
			}
		})
	}
}

func TestLeadingSpaces(t *testing.T) {
	cases := map[string]int{"": 0, "x": 0, "  x": 2, "    ": 4, "\t x": 0}
	for in, want := range cases {
		if got := leadingSpaces(in); got != want {
			t.Errorf("leadingSpaces(%q) = %d, want %d", in, got, want)
		}
	}
}
