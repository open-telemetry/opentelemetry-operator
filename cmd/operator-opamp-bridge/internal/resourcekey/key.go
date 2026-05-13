// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package resourcekey handles OpAMP config map keys that identify Kubernetes resources.
package resourcekey

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// KindOtelCol identifies OpenTelemetryCollector CRD configuration keys.
	KindOtelCol = "otelcol"
	// KindConfigMap identifies standalone ConfigMap configuration keys.
	KindConfigMap = "configmap"
)

// Key identifies a Kubernetes resource in OpAMP config maps.
type Key struct {
	name      string
	namespace string
	kind      string
}

// New creates a Key.
func New(namespace, name, kind string) Key {
	return Key{name: name, namespace: namespace, kind: kind}
}

// Parse creates a Key from an OpAMP config map key string.
func Parse(key string) (Key, error) {
	s := strings.Split(key, "/")
	// We expect map keys to be of the form name, namespace/name, or kind/namespace/name.
	switch len(s) {
	case 1:
		return New("", s[0], ""), nil
	case 2:
		return New(s[0], s[1], ""), nil
	case 3:
		return New(s[1], s[2], s[0]), nil
	default:
		return Key{}, errors.New("invalid key")
	}
}

// Name returns the Kubernetes resource name.
func (k Key) Name() string {
	return k.name
}

// Namespace returns the Kubernetes namespace.
func (k Key) Namespace() string {
	return k.namespace
}

// Kind returns the OpAMP resource key kind.
func (k Key) Kind() string {
	return k.kind
}

// String returns the OpAMP config map key string.
func (k Key) String() string {
	if k.kind != "" {
		return fmt.Sprintf("%s/%s/%s", k.kind, k.namespace, k.name)
	}
	if k.namespace == "" {
		return k.name
	}
	return fmt.Sprintf("%s/%s", k.namespace, k.name)
}
