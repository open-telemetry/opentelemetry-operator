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
