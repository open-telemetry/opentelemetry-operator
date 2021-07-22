package discovery

import (
	"context"
	"sort"
	"testing"

	gokitlog "github.com/go-kit/log"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/config"
	"github.com/stretchr/testify/assert"
)

func TestTargetDiscovery(t *testing.T) {
	cfg, err := config.Load("./testdata/test.yaml")
	assert.NoError(t, err)
	manager := NewManager(context.Background(), gokitlog.NewNopLogger())

	results := make(chan []string)
	manager.Watch(func(targets []allocation.TargetItem) {
		var result []string
		for _, t := range targets {
			result = append(result, t.TargetURL)
		}
		results <- result
	})

	t.Run("should discover targets", func(t *testing.T) {
		err := manager.ApplyConfig(cfg)
		assert.NoError(t, err)

		gotTargets := <-results
		wantTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000"}

		sort.Strings(gotTargets)
		sort.Strings(wantTargets)
		assert.Equal(t, gotTargets, wantTargets)
	})

	t.Run("should update targets", func(t *testing.T) {
		cfg.Config.ScrapeConfigs[0]["static_configs"] = []map[string]interface{}{
			{"targets": []string{"prom.domain:9004", "prom.domain:9005"}},
		}

		err := manager.ApplyConfig(cfg)
		assert.NoError(t, err)

		gotTargets := <-results
		wantTargets := []string{"prom.domain:9004", "prom.domain:9005", "promfile.domain:1001", "promfile.domain:3000"}

		sort.Strings(gotTargets)
		sort.Strings(wantTargets)
		assert.Equal(t, gotTargets, wantTargets)
	})
}
