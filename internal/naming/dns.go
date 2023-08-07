// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Additional copyrights:
// Copyright The Jaeger Authors

package naming

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var regex = regexp.MustCompile(`[a-z0-9]`)

// DNSName returns a dns-safe string for the given name.
// Any char that is not [a-z0-9] is replaced by "-" or "a".
// Replacement character "a" is used only at the beginning or at the end of the name.
// The function does not change length of the string.
// source: https://github.com/jaegertracing/jaeger-operator/blob/91e3b69ee5c8761bbda9d3cf431400a73fc1112a/pkg/util/dns_name.go#L15
func DNSName(name string) string {
	var d []rune

	for i, x := range strings.ToLower(name) {
		if regex.Match([]byte(string(x))) {
			d = append(d, x)
		} else {
			if i == 0 || i == utf8.RuneCountInString(name)-1 {
				d = append(d, 'a')
			} else {
				d = append(d, '-')
			}
		}
	}

	return string(d)
}
