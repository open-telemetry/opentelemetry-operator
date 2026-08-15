// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package instrumentation

import (
	"bytes"
	"embed"
	"log"
	"os"
	"testing"
	"text/template"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// This suite checks that an auto-instrumented workload really delivers telemetry.
// Each test deploys, into a namespace of its own, an otlp-sink
// (tests/test-e2e-apps/otlp-sink), a collector exporting to it, an Instrumentation
// CR and a sample app annotated for injection. It then verifies the webhook injected
// the agent, drives the app over the API server's pod proxy, and asserts on the
// telemetry the sink received: which spans and metrics arrived, and with which
// resource identity. Sharing nothing but the cluster, the tests run in parallel.
//
// On failure, e2e.DumpNamespaceOnFailure logs pod/container statuses, events,
// every container's trailing log (app, agent init containers, collector, sink)
// and the applied CRs; the sink assertion errors additionally report an
// inventory of the telemetry that WAS received, so a failure is diagnosable from
// the test output alone.

var (
	testenv   env.Environment
	sinkImage string
)

//go:embed testdata
var manifestFS embed.FS

// render reads an embedded manifest and executes it as a template with data.
// Manifests without template actions pass through unchanged; a missing key is a
// programming error and fails the render.
func render(name string, data map[string]string) string {
	b, err := manifestFS.ReadFile("testdata/" + name)
	if err != nil {
		panic(err)
	}
	tmpl := template.Must(template.New(name).Option("missingkey=error").Parse(string(b)))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestMain(m *testing.M) {
	// The Makefile derives this from TEST_E2E_APPS_IMG_PREFIX, which is also what
	// `make load-image-test-e2e-apps` tags a locally built sink with; keeping the
	// default there rather than here means one image name for the whole suite.
	sinkImage = os.Getenv("OTLPSINK_IMG")
	if sinkImage == "" {
		log.Fatal("OTLPSINK_IMG environment variable must be set (use `make e2e-instrumentation-go`)")
	}

	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("failed to parse e2e flags: %v", err)
	}
	// Initialize the shared klient once, up front: unlike the other Go e2e suites
	// this one runs its features in parallel, and they must not race on the lazy
	// initialization inside envconf.Config.Client.
	cfg.Client()
	testenv = env.NewWithConfig(cfg)
	os.Exit(testenv.Run(m))
}
