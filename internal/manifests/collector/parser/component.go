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

package parser

import (
	"fmt"
	"strings"
)

type ComponentType int

const (
	ComponentTypeReceiver ComponentType = iota
	ComponentTypeExporter
	ComponentTypeProcessor
	ComponentTypeConnector
)

func (c ComponentType) String() string {
	return [...]string{"receiver", "exporter", "processor", "connector"}[c]
}
func (c ComponentType) Plural() string {
	return fmt.Sprintf("%ss", c.String())
}

func ComponentName(name string) string {
	// processors have a name like:
	// - myprocessor/custom
	// - myprocessor
	// we extract the "myprocessor" part and see if we have a parser for the processor
	if strings.Contains(name, "/") {
		return name[:strings.Index(name, "/")]
	}

	return name
}
