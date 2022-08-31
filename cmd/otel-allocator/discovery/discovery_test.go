package discovery

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	gokitlog "github.com/go-kit/log"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var cfg config.Config
var manager *Manager
var results chan []string

func TestMain(m *testing.M) {
	var err error
	cfg, err = config.Load("./testdata/test.yaml")
	if err != nil {
		fmt.Printf("failed to load config file: %v", err)
		os.Exit(1)
	}
	manager = NewManager(ctrl.Log.WithName("test"), context.Background(), gokitlog.NewNopLogger())

	results = make(chan []string)
	manager.Watch(func(targets []allocation.TargetItem) {
		var result []string
		for _, t := range targets {
			result = append(result, t.TargetURL)
		}
		results <- result
	})

	code := m.Run()

	close(manager.close)

	os.Exit(code)
}

func TestTargetDiscovery(t *testing.T) {
	err := manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, &promconfig.Config{})
	assert.NoError(t, err)

	gotTargets := <-results
	wantTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000"}

	sort.Strings(gotTargets)
	sort.Strings(wantTargets)
	assert.Equal(t, gotTargets, wantTargets)
}

func TestTargetUpdate(t *testing.T) {
	cfg.Config.ScrapeConfigs[0].ServiceDiscoveryConfigs[1] = discovery.StaticConfig{
		{
			Targets: []model.LabelSet{
				{model.AddressLabel: "prom.domain:9004"},
				{model.AddressLabel: "prom.domain:9005"},
			},
			Labels: model.LabelSet{
				"my": "label",
			},
			Source: "0",
		},
	}

	err := manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, &promconfig.Config{})
	assert.NoError(t, err)

	gotTargets := <-results
	wantTargets := []string{"prom.domain:9004", "prom.domain:9005", "promfile.domain:1001", "promfile.domain:3000"}

	sort.Strings(gotTargets)
	sort.Strings(wantTargets)
	assert.Equal(t, gotTargets, wantTargets)
}
