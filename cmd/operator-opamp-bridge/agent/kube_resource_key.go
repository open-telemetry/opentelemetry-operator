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
