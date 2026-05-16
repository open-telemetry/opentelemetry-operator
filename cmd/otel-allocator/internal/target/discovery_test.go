// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/prometheus/prometheus/scrape"
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
			slices.Sort(gotTargets)
			slices.Sort(tt.want)
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
			for _, tgt := range gotTargets {
				h := tgt.Hash()
				assert.False(t, targetHashes[h], "Duplicate hash %d found for target %s (%s)", h, tgt.TargetURL, tgt.JobName)
				targetHashes[h] = true
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
	scu := &mockScrapeConfigUpdater{}
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(t, err)
	manager := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	d, err := NewDiscoverer(ctrl.Log.WithName("test"), manager, nil, scu, nil)
	require.NoError(t, err)
	d.processTargetGroups("test", groups, results, &promconfig.ScrapeConfig{JobName: "test"})

	// Verify all labels are in sorted order (the core invariant this test validates).
	// scrape.PopulateDiscoveredLabels adds scrape config defaults (__scheme__, etc.)
	// which interleave with the alphabetic test labels.
	var prevName string
	results[0].Labels.Range(func(l labels.Label) {
		if prevName != "" {
			assert.Less(t, prevName, l.Name, "labels should be sorted, but %q came after %q", l.Name, prevName)
		}
		prevName = l.Name
	})

	// Verify that all original group and target labels are present.
	for i := 'a'; i <= 'z'; i++ {
		name := string(rune(i))
		assert.Equal(t, name, results[0].Labels.Get(name), "expected label %q to be present", name)
	}
}

func TestPopulateDiscoveredLabels(t *testing.T) {
	tests := []struct {
		description string
		cfg         *promconfig.ScrapeConfig
		tLabels     model.LabelSet
		tgLabels    model.LabelSet
		wantLabels  map[string]string
	}{
		{
			description: "scrape config defaults and params are applied",
			cfg: &promconfig.ScrapeConfig{
				JobName:        "my-job",
				ScrapeInterval: model.Duration(30 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				MetricsPath:    "/metrics",
				Scheme:         "http",
				Params: url.Values{
					"module": []string{"http_2xx"},
				},
			},
			tLabels:  model.LabelSet{"__address__": "localhost:9090"},
			tgLabels: model.LabelSet{},
			wantLabels: map[string]string{
				"__address__":         "localhost:9090",
				"job":                 "my-job",
				"__scrape_interval__": "30s",
				"__scrape_timeout__":  "10s",
				"__metrics_path__":    "/metrics",
				"__scheme__":          "http",
				"__param_module":      "http_2xx",
			},
		},
		{
			description: "target labels override group labels",
			cfg: &promconfig.ScrapeConfig{
				JobName:     "job1",
				MetricsPath: "/metrics",
				Scheme:      "http",
			},
			tLabels:  model.LabelSet{"__address__": "target-addr", "env": "target-env"},
			tgLabels: model.LabelSet{"env": "group-env", "region": "us-west"},
			wantLabels: map[string]string{
				"__address__": "target-addr",
				"env":         "target-env",
				"region":      "us-west",
				"job":         "job1",
			},
		},
		{
			description: "existing labels are not overridden by scrape config",
			cfg: &promconfig.ScrapeConfig{
				JobName:     "default-job",
				MetricsPath: "/default-metrics",
				Scheme:      "https",
			},
			tLabels: model.LabelSet{
				"__address__":      "addr",
				"job":              "custom-job",
				"__metrics_path__": "/custom-path",
			},
			tgLabels: model.LabelSet{},
			wantLabels: map[string]string{
				"__address__":      "addr",
				"job":              "custom-job",
				"__metrics_path__": "/custom-path",
				"__scheme__":       "https",
			},
		},
	}

	groupBuilder := labels.NewScratchBuilder(labelBuilderPreallocSize)
	lb := labels.NewBuilder(labels.EmptyLabels())
	prometheusLabelsBuilder := labels.NewBuilder(labels.EmptyLabels())
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			groupLabels := populateGroupLabels(&groupBuilder, tc.tgLabels)
			lset := populateDiscoveredLabels(lb, newDiscoveredLabelDefaults(tc.cfg), tc.tLabels, groupLabels)

			scrape.PopulateDiscoveredLabels(prometheusLabelsBuilder, tc.cfg, tc.tLabels, tc.tgLabels)
			prometheusLabels := prometheusLabelsBuilder.Labels()
			assert.Equal(t, prometheusLabels, lset)
			for k, v := range tc.wantLabels {
				assert.Equal(t, v, lset.Get(k), "label %q mismatch", k)
			}
		})
	}
}

func BenchmarkProcessTargetGroups(b *testing.B) {
	tests := []struct {
		name             string
		groupCount       int
		targetsPerGroup  int
		groupLabelCount  int
		targetLabelCount int
	}{
		{
			name:             "1k_targets_10_labels",
			groupCount:       10,
			targetsPerGroup:  100,
			groupLabelCount:  5,
			targetLabelCount: 5,
		},
		{
			name:             "10k_targets_20_labels",
			groupCount:       100,
			targetsPerGroup:  100,
			groupLabelCount:  10,
			targetLabelCount: 10,
		},
	}

	scu := &mockScrapeConfigUpdater{}
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(registry)
	require.NoError(b, err)
	manager := discovery.NewManager(ctx, config.NopLogger, registry, sdMetrics)
	d, err := NewDiscoverer(ctrl.Log.WithName("benchmark"), manager, nil, scu, nil)
	require.NoError(b, err)

	cfg := &promconfig.ScrapeConfig{
		JobName:        "benchmark",
		ScrapeInterval: model.Duration(30 * time.Second),
		ScrapeTimeout:  model.Duration(10 * time.Second),
		MetricsPath:    "/metrics",
		Scheme:         "http",
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			groups := benchmarkTargetGroups(tt.groupCount, tt.targetsPerGroup, tt.groupLabelCount, tt.targetLabelCount)
			results := make([]*Item, tt.groupCount*tt.targetsPerGroup)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				d.processTargetGroups("benchmark", groups, results, cfg)
			}
		})
	}
}

func benchmarkTargetGroups(groupCount, targetsPerGroup, groupLabelCount, targetLabelCount int) []*targetgroup.Group {
	groups := make([]*targetgroup.Group, groupCount)
	for groupIndex := range groupCount {
		groupLabels := make(model.LabelSet, groupLabelCount)
		for labelIndex := range groupLabelCount {
			groupLabels[model.LabelName(fmt.Sprintf("__meta_kubernetes_group_label_%02d", labelIndex))] = model.LabelValue(fmt.Sprintf("group-%d-%d", groupIndex, labelIndex))
		}

		targets := make([]model.LabelSet, targetsPerGroup)
		for targetIndex := range targetsPerGroup {
			targetLabels := make(model.LabelSet, targetLabelCount+1)
			targetLabels[model.AddressLabel] = model.LabelValue(fmt.Sprintf("10.%d.%d.%d:9100", groupIndex%255, targetIndex/255, targetIndex%255))
			for labelIndex := range targetLabelCount {
				targetLabels[model.LabelName(fmt.Sprintf("__meta_kubernetes_target_label_%02d", labelIndex))] = model.LabelValue(fmt.Sprintf("target-%d-%d-%d", groupIndex, targetIndex, labelIndex))
			}
			targets[targetIndex] = targetLabels
		}

		groups[groupIndex] = &targetgroup.Group{
			Targets: targets,
			Labels:  groupLabels,
			Source:  fmt.Sprintf("benchmark/%d", groupIndex),
		}
	}
	return groups
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

	for i := range numConfigs {
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
