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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", SplunkOtelAgent())
}

func TestVersionFromBuild(t *testing.T) {
	// prepare
	otelCol = "0.0.2" // set during the build
	defer func() {
		otelCol = ""
	}()

	assert.Equal(t, otelCol, SplunkOtelAgent())
	assert.Contains(t, Get().String(), otelCol)
}

func TestTargetAllocatorFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", TargetAllocator())
}

func TestTargetAllocatorVersionFromBuild(t *testing.T) {
	// prepare
	targetAllocator = "0.0.2" // set during the build
	defer func() {
		targetAllocator = ""
	}()

	assert.Equal(t, targetAllocator, TargetAllocator())
	assert.Contains(t, Get().String(), targetAllocator)
}
