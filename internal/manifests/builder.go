// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifests

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder[Params any] func(params Params) ([]client.Object, error)

type ManifestFactory[T client.Object, Params any] func(params Params) (T, error)
type ManifestSliceFactory[T ~[]client.Object, Params any] func(params Params) (T, error)
type SimpleManifestFactory[T client.Object, Params any] func(params Params) T
type K8sManifestFactory[Params any] ManifestFactory[client.Object, Params]
type K8sManifestSliceFactory[Params any] ManifestSliceFactory[[]client.Object, Params]

func FactoryWithoutError[T client.Object, Params any](f SimpleManifestFactory[T, Params]) K8sManifestFactory[Params] {
	return func(params Params) (client.Object, error) {
		return f(params), nil
	}
}

func Factory[T client.Object, Params any](f ManifestFactory[T, Params]) K8sManifestFactory[Params] {
	return func(params Params) (client.Object, error) {
		return f(params)
	}
}

func FactorySlice[T []client.Object, Params any](f ManifestSliceFactory[T, Params]) K8sManifestSliceFactory[Params] {
	return func(params Params) ([]client.Object, error) {
		return f(params)
	}
}

// ObjectIsNotNil ensures that we only create an object IFF it isn't nil,
// and it's concrete type isn't nil either. This works around the Go type system
// by using reflection to verify its concrete type isn't nil.
func ObjectIsNotNil(obj client.Object) bool {
	return obj != nil && !reflect.ValueOf(obj).IsNil()
}
