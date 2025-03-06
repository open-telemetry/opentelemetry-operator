// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import "net/http"

type Headers map[string]string

func (h Headers) ToHTTPHeader() http.Header {
	newMap := make(map[string][]string)
	for key, value := range h {
		newMap[key] = []string{value}
	}
	return newMap
}
