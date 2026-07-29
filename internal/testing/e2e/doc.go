// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package e2e is a small framework for writing full-deployment end-to-end tests
// in Go that need semantic checks (e.g. "the right metric, with the right labels
// and value, made it end-to-end") which are awkward to express in chainsaw/bash.
package e2e
