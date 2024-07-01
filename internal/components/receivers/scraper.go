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

package receivers

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

var (
	_ components.ComponentPortParser = &ScraperParser{}
)

type ScraperParser struct {
	componentType string
}

func (s *ScraperParser) Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error) {
	return nil, nil
}

func (s *ScraperParser) ParserType() string {
	return s.componentType
}

func (s *ScraperParser) ParserName() string {
	return fmt.Sprintf("__%s", s.componentType)
}

func NewScraperParser(name string) *ScraperParser {
	return &ScraperParser{
		componentType: components.ComponentType(name),
	}
}
