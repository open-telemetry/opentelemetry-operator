// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package e2e is a small framework for writing full-deployment end-to-end tests
// in Go that need semantic checks (e.g. "the right metric, with the right labels
// and value, made it end-to-end") which are awkward to express in chainsaw/bash.
//
// The helpers (build-tagged "e2e") lean on sigs.k8s.io/e2e-framework for the test
// lifecycle and on controller-runtime / client-go for cluster operations — manifests
// are server-side-applied as unstructured objects, and Prometheus is queried over the
// API server's service proxy (no port-forward). They use only dependencies the
// operator already builds against, so the e2e tests add nothing to its module
// footprint, and individual tests carry almost no bespoke Kubernetes plumbing.
package e2e
