// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"errors"
	"fmt"
	"strings"
)

type kubeResourceKey struct {
	name      string
	namespace string
}

func newKubeResourceKey(namespace string, name string) kubeResourceKey {
	return kubeResourceKey{name: name, namespace: namespace}
}

func kubeResourceFromKey(key string) (kubeResourceKey, error) {
	s := strings.Split(key, "/")
	// We expect map keys to be of the form name/namespace
	if len(s) != 2 {
		return kubeResourceKey{}, errors.New("invalid key")
	}
	return newKubeResourceKey(s[0], s[1]), nil
}

func (k kubeResourceKey) String() string {
	return fmt.Sprintf("%s/%s", k.namespace, k.name)
}
