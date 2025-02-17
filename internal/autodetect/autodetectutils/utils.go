// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetectutils

import (
	"fmt"
	"os"
)

const (
	SA_ENV_VAR          = "SERVICE_ACCOUNT_NAME"
	NAMESPACE_ENV_VAR   = "NAMESPACE"
	NAMESPACE_FILE_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func GetOperatorNamespace() (string, error) {
	namespace := os.Getenv(NAMESPACE_ENV_VAR)
	if namespace != "" {
		return namespace, nil
	}

	nsBytes, err := os.ReadFile(NAMESPACE_FILE_PATH)
	if err != nil {
		return "", err
	}
	return string(nsBytes), nil
}

func GetOperatorServiceAccount() (string, error) {
	sa := os.Getenv(SA_ENV_VAR)
	if sa == "" {
		return sa, fmt.Errorf("%s env variable not found", SA_ENV_VAR)
	}
	return sa, nil
}
