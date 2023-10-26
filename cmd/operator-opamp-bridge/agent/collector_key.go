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

type collectorKey struct {
	name      string
	namespace string
}

func newCollectorKey(namespace string, name string) collectorKey {
	return collectorKey{name: name, namespace: namespace}
}

func collectorKeyFromKey(key string) (collectorKey, error) {
	s := strings.Split(key, "/")
	// We expect map keys to be of the form name/namespace
	if len(s) != 2 {
		return collectorKey{}, errors.New("invalid key")
	}
	return newCollectorKey(s[0], s[1]), nil
}

func (k collectorKey) String() string {
	return fmt.Sprintf("%s/%s", k.namespace, k.name)
}
