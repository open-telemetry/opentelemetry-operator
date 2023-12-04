package config

import "net/http"

type Headers map[string]string

func (h Headers) ToHttpHeader() http.Header {
	newMap := make(map[string][]string)
	for key, value := range h {
		newMap[key] = []string{value}
	}
	return newMap
}
