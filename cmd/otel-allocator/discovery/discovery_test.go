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

package discovery

import (
	"context"
	"sort"
	"testing"

	gokitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

func TestDiscovery(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "base case",
			args: args{
				file: "./testdata/test.yaml",
			},
			want: []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000"},
		},
		{
			name: "update",
			args: args{
				file: "./testdata/test_update.yaml",
			},
			want: []string{"prom.domain:9004", "prom.domain:9005", "promfile.domain:1001", "promfile.domain:3000"},
		},
	}
	manager := NewManager(context.Background(), ctrl.Log.WithName("test"), gokitlog.NewNopLogger(), nil)
	defer close(manager.close)

	results := make(chan []string)
	manager.Watch(func(targets map[string]*target.Item) {
		var result []string
		for _, t := range targets {
			result = append(result, t.TargetURL[0])
		}
		results <- result
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(tt.args.file)
			assert.NoError(t, err)
			assert.True(t, len(cfg.Config.ScrapeConfigs) > 0)
			err = manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, cfg.Config)
			assert.NoError(t, err)

			gotTargets := <-results
			sort.Strings(gotTargets)
			sort.Strings(tt.want)
			assert.Equal(t, tt.want, gotTargets)
		})
	}
}
