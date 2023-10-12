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

package manifests

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder func(params Params) ([]client.Object, error)

type ManifestFactory[T client.Object] func(params Params) (T, error)
type SimpleManifestFactory[T client.Object] func(params Params) T
type K8sManifestFactory ManifestFactory[client.Object]

func FactoryWithoutError[T client.Object](f SimpleManifestFactory[T]) K8sManifestFactory {
	return func(params Params) (client.Object, error) {
		return f(params), nil
	}
}

func Factory[T client.Object](f ManifestFactory[T]) K8sManifestFactory {
	return func(params Params) (client.Object, error) {
		return f(params)
	}
}

// ObjectIsNotNil ensures that we only create an object IFF it isn't nil,
// and it's concrete type isn't nil either. This works around the Go type system
// by using reflection to verify its concrete type isn't nil.
func ObjectIsNotNil(obj client.Object) bool {
	return obj != nil && !reflect.ValueOf(obj).IsNil()
}
