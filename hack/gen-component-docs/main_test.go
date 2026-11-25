// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestRender(t *testing.T) {
	out := render([]section{
		{title: "Receivers", parsers: []components.Parser{
			components.NewBuilder[any]().WithName("zeta").MustBuild(),
			components.NewBuilder[any]().WithName("alpha_beta").WithAlias("alphabeta").MustBuild(),
		}},
		{title: "Exporters", parsers: nil},
	})

	assert.True(t, strings.HasPrefix(out, header))
	assert.Contains(t, out, "\n## Receivers\n\n"+
		"| Component | Also accepted as |\n"+
		"| --------- | ---------------- |\n"+
		"| `alpha_beta` | `alphabeta` |\n"+
		"| `zeta` |  |\n")
	assert.Contains(t, out, "\n## Exporters\n\n"+
		"| Component | Also accepted as |\n"+
		"| --------- | ---------------- |\n")
}
