// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// PortName defines the port name used in services, ingresses and routes.
// The port name in pod and ingress spec has to be maximum 15 characters long.
func PortName(receiverName string, port int32) string {
	if len(receiverName) > 15 {
		return fmt.Sprintf("port-%d", port)
	}

	candidate := strings.ReplaceAll(receiverName, "/", "-")
	candidate = strings.ReplaceAll(candidate, "_", "-")

	if len(validation.IsValidPortName(candidate)) > 0 {
		return fmt.Sprintf("port-%d", port)
	}

	// matches the pattern and has less than 15 chars -- the candidate name is good to go!
	return candidate
}
