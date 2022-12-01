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

// Package config contains the operator's runtime configuration.
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeHandler(t *testing.T) {
	// prepare
	internal := 0
	callback := func() error {
		internal += 1
		return nil
	}
	h := newOnChange()

	h.Register(callback)

	for i := 0; i < 5; i++ {
		assert.Equal(t, i, internal)
		require.NoError(t, h.Do())
		assert.Equal(t, i+1, internal)
	}
}
