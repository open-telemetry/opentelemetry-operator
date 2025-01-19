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

package prehook

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

type myHook struct {
}

func (m myHook) Apply(_ map[string]*target.Item) map[string]*target.Item {
	panic("implement me")
}

func (m myHook) SetConfig(_ map[string][]*relabel.Config) {
	panic("implement me")
}

func (m myHook) GetConfig() map[string][]*relabel.Config {
	panic("implement me")
}

func TestRegister(t *testing.T) {
	myProvider := func(log logr.Logger) Hook {
		return myHook{}
	}

	require.NoError(t, Register("foo", myProvider))

	hook := New("foo", logr.New(log.NullLogSink{}))
	require.NotNil(t, hook)
}

func TestRegisterMissing(t *testing.T) {
	hook := New("bar", logr.New(log.NullLogSink{}))
	require.Nil(t, hook)
}
