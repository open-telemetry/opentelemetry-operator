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

package naming

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// DNS_LABEL constraints: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
	dnsLabelValidation = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
)

// PortName defines the port name used in services, ingresses and routes.
// The port name in pod and ingress spec has to be maximum 15 characters long.
func PortName(receiverName string, port int32) string {
	if len(receiverName) > 15 {
		return fmt.Sprintf("port-%d", port)
	}

	candidate := strings.ReplaceAll(receiverName, "/", "-")
	candidate = strings.ReplaceAll(candidate, "_", "-")

	if !dnsLabelValidation.MatchString(candidate) {
		return fmt.Sprintf("port-%d", port)
	}

	// matches the pattern and has less than 15 chars -- the candidate name is good to go!
	return candidate
}
