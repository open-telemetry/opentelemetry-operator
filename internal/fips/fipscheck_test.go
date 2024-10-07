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

package fips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFipsCheck(t *testing.T) {
	fipsCheck := NewFipsCheck([]string{"rec1", "rec2"}, []string{"exp1"}, []string{"processor"}, []string{"ext1"})
	blocked := fipsCheck.DisabledComponents(
		map[string]interface{}{"otlp": true, "rec1/my": true},
		map[string]interface{}{"exp1": true},
		map[string]interface{}{"processor": true},
		map[string]interface{}{"ext1": true})

	assert.Equal(t, []string{"rec1", "exp1", "processor", "ext1"}, blocked)
}
