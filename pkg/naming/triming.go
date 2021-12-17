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
	"fmt"
	"regexp"
)

var regexpEndReplace, regexpBeginReplace *regexp.Regexp

func init() {
	regexpEndReplace, _ = regexp.Compile("[^A-Za-z0-9]+$")
	regexpBeginReplace, _ = regexp.Compile("^[^A-Za-z0-9]+")
}

// Truncate will shorten the length of the instance name so that it contains at most max chars when combined with the fixed part
// If the fixed part is already bigger than the max, this function is noop.
// source: https://github.com/jaegertracing/jaeger-operator/blob/91e3b69ee5c8761bbda9d3cf431400a73fc1112a/pkg/util/truncate.go#L17
func Truncate(format string, max int, values ...interface{}) string {
	var truncated []interface{}
	result := fmt.Sprintf(format, values...)
	if excess := len(result) - max; excess > 0 {
		// we try to reduce the first string we find
		for _, value := range values {
			if excess == 0 {
				truncated = append(truncated, value)
				continue
			}

			if s, ok := value.(string); ok {
				if len(s) > excess {
					value = s[:len(s)-excess]
					excess = 0
				} else {
					value = "" // skip this value entirely
					excess = excess - len(s)
				}
			}

			truncated = append(truncated, value)
		}
		result = fmt.Sprintf(format, truncated...)
	}

	// if at this point, the result is still bigger than max, apply a hard cap:
	if len(result) > max {
		return result[:max]
	}

	return trimNonAlphaNumeric(result)
}

// trimNonAlphaNumeric remove all non-alphanumeric values from start and end of the string
// source: https://github.com/jaegertracing/jaeger-operator/blob/91e3b69ee5c8761bbda9d3cf431400a73fc1112a/pkg/util/truncate.go#L53
func trimNonAlphaNumeric(text string) string {
	newText := regexpEndReplace.ReplaceAllString(text, "")
	return regexpBeginReplace.ReplaceAllString(newText, "")
}
