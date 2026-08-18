// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"errors"
	"fmt"
	"strings"
)

type KubeResourceKey struct {
	name      string
	namespace string
}

func NewKubeResourceKey(namespace, name string) KubeResourceKey {
	return KubeResourceKey{name: name, namespace: namespace}
}

func kubeResourceFromKey(key string) (KubeResourceKey, error) {
	s := strings.Split(key, "/")
	// We expect map keys to be of the form namespace/name.
	if len(s) != 2 {
		return KubeResourceKey{}, errors.New("invalid key")
	}
	return NewKubeResourceKey(s[0], s[1]), nil
}

func (k KubeResourceKey) String() string {
	return fmt.Sprintf("%s/%s", k.namespace, k.name)
}
