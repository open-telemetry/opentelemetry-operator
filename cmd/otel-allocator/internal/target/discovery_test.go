// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"context"
	"errors"
	"hash"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

var defaultScrapeProtocols = []promconfig.ScrapeProtocol{
	promconfig.OpenMetricsText1_0_0,
	promconfig.OpenMetricsText0_0_1,
	promconfig.PrometheusText0_0_4,
}

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
			want: []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "prom.domain:8001", "promfile.domain:1001", "promfile.domain:3000"},
		},
		{
			name: "update",
			args: args{
				file: "./testdata/test_update.yaml",
			},
			want: []string{"prom.domain:9004", "prom.domain:9005", "promfile.domain:1001", "promfile.domain:3000"},
		},
	}
	scu := &mockScrapeConfigUpdater{}
	ctx, cancelFunc := context.WithCancel(context.Background())
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(t, err)
	d := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	results := make(chan []string)
	manager, err := NewDiscoverer(ctrl.Log.WithName("test"), d, nil, scu, func(targets []*Item) {
		var result []string
		for _, t := range targets {
			result = append(result, t.TargetURL)
		}
		results <- result
	})
	require.NoError(t, err)

	defer func() { manager.Close() }()
	defer cancelFunc()

	go func() {
		runErr := d.Run()
		assert.Error(t, runErr)
	}()
	go func() {
		runErr := manager.Run()
		assert.NoError(t, runErr)
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.CreateDefaultConfig()
			err := config.LoadFromFile(tt.args.file, &cfg)
			assert.NoError(t, err)
			assert.True(t, len(cfg.PromConfig.ScrapeConfigs) > 0)
			err = manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, cfg.PromConfig.ScrapeConfigs)
			assert.NoError(t, err)

			gotTargets := <-results
			sort.Strings(gotTargets)
			sort.Strings(tt.want)
			assert.Equal(t, tt.want, gotTargets)

			// check the updated scrape configs
			expectedScrapeConfigs := map[string]*promconfig.ScrapeConfig{}
			for _, scrapeConfig := range cfg.PromConfig.ScrapeConfigs {
				expectedScrapeConfigs[scrapeConfig.JobName] = scrapeConfig
			}
			assert.Equal(t, expectedScrapeConfigs, scu.mockCfg)
		})
	}
}

func TestDiscovery_ScrapeConfigHashing(t *testing.T) {
	// these tests are meant to be run sequentially in this order, to test
	// that hashing doesn't cause us to send the wrong information.
	tests := []struct {
		description string
		cfg         *promconfig.Config
		expectErr   bool
	}{
		{
			description: "base config",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/0",
						HonorTimestamps: true,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.*)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "different bool",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/0",
						HonorTimestamps: false,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.*)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "different job name",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/1",
						HonorTimestamps: false,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.*)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "different key",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/1",
						HonorTimestamps: false,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.*)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "unset scrape interval",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/1",
						HonorTimestamps: false,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.*)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "different regex",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/testapp/testapp/1",
						HonorTimestamps: false,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						MetricsPath:     "/metrics",
						Scheme:          "http",
						HTTPClientConfig: commonconfig.HTTPClientConfig{
							FollowRedirects: true,
						},
						RelabelConfigs: []*relabel.Config{
							{
								SourceLabels: model.LabelNames{model.LabelName("job")},
								Separator:    ";",
								Regex:        relabel.MustNewRegexp("(.+)"),
								TargetLabel:  "__tmp_prometheus_job_name",
								Replacement:  "$$1",
								Action:       relabel.Replace,
							},
						},
					},
				},
			},
		},
		{
			description: "mock error on update - no hash update",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName: "error",
					},
				},
			},
			expectErr: true,
		},
	}
	var (
		lastValidHash   hash.Hash
		expectedConfig  map[string]*promconfig.ScrapeConfig
		lastValidConfig map[string]*promconfig.ScrapeConfig
	)

	scu := &mockScrapeConfigUpdater{}
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(t, err)
	d := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	manager, err := NewDiscoverer(ctrl.Log.WithName("test"), d, nil, scu, nil)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			err := manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, tc.cfg.ScrapeConfigs)
			if !tc.expectErr {
				expectedConfig = make(map[string]*promconfig.ScrapeConfig)
				for _, configs := range manager.configsMap {
					for _, scrapeConfig := range configs {
						expectedConfig[scrapeConfig.JobName] = scrapeConfig
					}
				}
				assert.NoError(t, err)
				assert.NotZero(t, manager.scrapeConfigsHash)
				// Assert that scrape configs in manager are correctly
				// reflected in the scrape job updater.
				assert.Equal(t, expectedConfig, scu.mockCfg)

				lastValidHash = manager.scrapeConfigsHash
				lastValidConfig = expectedConfig
			} else {
				// In case of error, assert that we retain the last
				// known valid config.
				assert.Error(t, err)
				assert.Equal(t, lastValidHash, manager.scrapeConfigsHash)
				assert.Equal(t, lastValidConfig, scu.mockCfg)
			}

		})
	}
}

func TestDiscoveryTargetHashing(t *testing.T) {
	tests := []struct {
		description string
		cfg         *promconfig.Config
	}{
		{
			description: "same targets in two different jobs",
			cfg: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "prometheus",
						HonorTimestamps: true,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics",
						Scheme:          "http",
						ServiceDiscoveryConfigs: discovery.Configs{
							discovery.StaticConfig{
								{
									Targets: []model.LabelSet{
										{"__address__": "prom.domain:9001"},
										{"__address__": "prom.domain:9002"},
										{"__address__": "prom.domain:9003"},
									},
								},
							},
						},
					},
					{
						JobName:         "prometheus2",
						HonorTimestamps: true,
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeProtocols: defaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(30 * time.Second),
						MetricsPath:     "/metrics2",
						Scheme:          "http",
						ServiceDiscoveryConfigs: discovery.Configs{
							discovery.StaticConfig{
								{
									Targets: []model.LabelSet{
										{"__address__": "prom.domain:9001"},
										{"__address__": "prom.domain:9002"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	scu := &mockScrapeConfigUpdater{}
	ctx, cancelFunc := context.WithCancel(context.Background())
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(t, err)
	d := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	results := make(chan []*Item)
	manager, err := NewDiscoverer(ctrl.Log.WithName("test"), d, nil, scu, func(targets []*Item) {
		var result []*Item
		result = append(result, targets...)
		results <- result
	})
	require.NoError(t, err)

	defer manager.Close()
	defer cancelFunc()

	go func() {
		runErr := d.Run()
		assert.Error(t, runErr)
	}()
	go func() {
		runErr := manager.Run()
		assert.NoError(t, runErr)
	}()

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			assert.NoError(t, err)
			assert.True(t, len(tt.cfg.ScrapeConfigs) > 0)
			err = manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, tt.cfg.ScrapeConfigs)
			assert.NoError(t, err)

			gotTargets := <-results

			// Verify that all targets have different hashes
			targetHashes := make(map[ItemHash]bool)
			for _, target := range gotTargets {
				hash := target.Hash()
				if _, exists := targetHashes[hash]; exists {
					t.Errorf("Duplicate hash %d found for target %s (%s)", hash, target.TargetURL, target.JobName)
				}
				targetHashes[hash] = true
			}
			assert.Equal(t, len(gotTargets), len(targetHashes), "Number of unique hashes should match number of targets")
		})
	}
}

func TestDiscovery_NoConfig(t *testing.T) {
	scu := &mockScrapeConfigUpdater{mockCfg: map[string]*promconfig.ScrapeConfig{}}
	ctx, cancelFunc := context.WithCancel(context.Background())
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(t, err)
	d := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	manager, err := NewDiscoverer(ctrl.Log.WithName("test"), d, nil, scu, nil)
	require.NoError(t, err)
	defer close(manager.close)
	defer cancelFunc()

	go func() {
		err := d.Run()
		assert.Error(t, err)
	}()
	// check the updated scrape configs
	expectedScrapeConfigs := map[string]*promconfig.ScrapeConfig{}
	assert.Equal(t, expectedScrapeConfigs, scu.mockCfg)
}

func TestProcessTargetGroups_StableLabelIterationOrder(t *testing.T) {
	// Labels are used to compute the hash of targets and hashing is
	// reliant on consistent ordering of labels. Creating one label
	// per letter of the english alphabet is enough to reliably
	// reproduce inconsistent label ordering due to non-deterministic
	// map iteration order + lack of sorting.
	groupLabels := model.LabelSet{}
	for i := 'a'; i <= 'm'; i++ {
		groupLabels[model.LabelName(i)] = model.LabelValue(i)
	}

	targetLabels := model.LabelSet{}
	for i := 'n'; i <= 'z'; i++ {
		targetLabels[model.LabelName(i)] = model.LabelValue(i)
	}

	groups := []*targetgroup.Group{
		{
			Labels:  groupLabels,
			Source:  "",
			Targets: []model.LabelSet{targetLabels},
		},
	}

	results := make([]*Item, 1)
	var d *Discoverer
	d.processTargetGroups("test", groups, results)

	for i, l := range results[0].Labels {
		expected := string(rune('a' + i))
		assert.Equal(t, expected, l.Name, "unexpected label key at index %d", i)
		assert.Equal(t, expected, l.Value, "unexpected label value at index %d", i)
	}
}

func BenchmarkApplyScrapeConfig(b *testing.B) {
	numConfigs := 1000
	scrapeConfig := promconfig.ScrapeConfig{
		JobName:         "serviceMonitor/testapp/testapp/0",
		HonorTimestamps: true,
		ScrapeInterval:  model.Duration(30 * time.Second),
		ScrapeTimeout:   model.Duration(30 * time.Second),
		MetricsPath:     "/metrics",
		Scheme:          "http",
		HTTPClientConfig: commonconfig.HTTPClientConfig{
			FollowRedirects: true,
		},
		RelabelConfigs: []*relabel.Config{
			{
				SourceLabels: model.LabelNames{model.LabelName("job")},
				Separator:    ";",
				Regex:        relabel.MustNewRegexp("(.*)"),
				TargetLabel:  "__tmp_prometheus_job_name",
				Replacement:  "$$1",
				Action:       relabel.Replace,
			},
		},
	}
	cfg := &promconfig.Config{
		ScrapeConfigs: make([]*promconfig.ScrapeConfig, numConfigs),
	}

	for i := 0; i < numConfigs; i++ {
		cfg.ScrapeConfigs[i] = &scrapeConfig
	}

	scu := &mockScrapeConfigUpdater{}
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(b, err)
	d := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	manager, err := NewDiscoverer(ctrl.Log.WithName("test"), d, nil, scu, nil)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, cfg.ScrapeConfigs)
		require.NoError(b, err)
	}
}

var _ scrapeConfigsUpdater = &mockScrapeConfigUpdater{}

// mockScrapeConfigUpdater is a mock implementation of the scrapeConfigsUpdater.
// If a job with name "error" is provided to the UpdateScrapeConfigResponse,
// it will return an error for testing purposes.
type mockScrapeConfigUpdater struct {
	mockCfg map[string]*promconfig.ScrapeConfig
}

func (m *mockScrapeConfigUpdater) UpdateScrapeConfigResponse(cfg map[string]*promconfig.ScrapeConfig) error {
	if _, ok := cfg["error"]; ok {
		return errors.New("error")
	}

	m.mockCfg = cfg
	return nil
}
