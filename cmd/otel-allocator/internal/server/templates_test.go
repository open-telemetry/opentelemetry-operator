// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tmplBody = `
        <p>{{.Index|even}}</p>
`

const want = `
        <p>true</p>
`

type testFuncsInput struct {
	Index int
}

var tmpl = template.Must(template.New("countTest").Funcs(templateFunctions).Parse(tmplBody))

func TestTemplateFuncs(t *testing.T) {
	buf := new(bytes.Buffer)
	input := testFuncsInput{
		Index: 32,
	}
	require.NoError(t, tmpl.Execute(buf, input))
	assert.EqualValues(t, want, buf.String())
}

func TestNoCrash(t *testing.T) {
	buf := new(bytes.Buffer)
	assert.NotPanics(t, func() { WriteHTMLPageHeader(buf, HeaderData{Title: "Foo"}) })
	assert.NotPanics(t, func() {
		WriteHTMLPropertiesTable(buf, PropertiesTableData{Headers: []string{"foo"}, Rows: [][]Cell{{NewCell("bar")}}})
	})
	assert.NotPanics(t, func() { WriteHTMLPageFooter(buf) })
}
