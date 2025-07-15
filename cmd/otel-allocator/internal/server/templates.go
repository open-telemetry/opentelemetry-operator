// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	_ "embed"
	"html/template"
	"io"
	"log"
)

var (
	templateFunctions = template.FuncMap{
		"even": even,
	}

	//go:embed templates/page_header.html
	headerBytes    []byte
	headerTemplate = parseTemplate("header", headerBytes)

	//go:embed templates/page_footer.html
	footerBytes    []byte
	footerTemplate = parseTemplate("footer", footerBytes)

	//go:embed templates/properties_table.html
	propertiesTableBytes    []byte
	propertiesTableTemplate = parseTemplate("properties_table", propertiesTableBytes)

	//go:embed templates/bad_request.html
	badRequestBytes    []byte
	badRequestTemplate = parseTemplate("bad_request", badRequestBytes)

	//go:embed templates/not_found.html
	notFoundBytes    []byte
	notFoundTemplate = parseTemplate("not_found", notFoundBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(template.New(name).Funcs(templateFunctions).Parse(string(bytes)))
}

// HeaderData contains data for the header template.
type HeaderData struct {
	Title string
}

// BadRequestData contains data for the bad request template.
type BadRequestData struct {
	Error   string
	Example string
}

// BadRequestData contains data for the bad request template.
type NotFoundData struct {
	ResourceType string
	ResourceName string
}

// WriteHTMLPageHeader writes the header.
func WriteHTMLPageHeader(w io.Writer, hd HeaderData) {
	if err := headerTemplate.Execute(w, hd); err != nil {
		log.Printf("ta: executing template: %v", err)
	}
}

// PropertiesTableData contains data for properties table template.
type PropertiesTableData struct {
	Headers []string
	Rows    [][]Cell
}

// Cell represents a cell in a row.
type Cell struct {
	// Link is the URL to link to. If empty, no link is created.
	Link string
	// Text is the text to display in the cell.
	Text string
	// Preformatted indicates if the text should be displayed as preformatted text.
	Preformatted bool
}

func NewCell(text string) Cell {
	return Cell{Text: text}
}

func Text(text string) Cell {
	return Cell{Text: text}
}

// WriteHTMLPropertiesTable writes the HTML for properties table.
func WriteHTMLPropertiesTable(w io.Writer, chd PropertiesTableData) {
	if err := propertiesTableTemplate.Execute(w, chd); err != nil {
		log.Printf("ta: executing template: %v", err)
	}
}

// WriteHTMLPageFooter writes the footer.
func WriteHTMLPageFooter(w io.Writer) {
	if err := footerTemplate.Execute(w, nil); err != nil {
		log.Printf("ta: executing template: %v", err)
	}
}

// WriteHTMLBadRequest writes the bad request page.
func WriteHTMLBadRequest(w io.Writer, br BadRequestData) {
	if err := badRequestTemplate.Execute(w, br); err != nil {
		log.Printf("ta: executing template: %v", err)
	}
}

// WriteHTMLNotFound writes the not found page.
func WriteHTMLNotFound(w io.Writer, nf NotFoundData) {
	if err := notFoundTemplate.Execute(w, nf); err != nil {
		log.Printf("ta: executing template: %v", err)
	}
}

func even(x int) bool {
	return x%2 == 0
}
