// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

func BenchmarkServerTargetsHandler(b *testing.B) {
	random := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
	var table = []struct {
		numCollectors int
		numJobs       int
	}{
		{numCollectors: 100, numJobs: 100},
		{numCollectors: 100, numJobs: 1000},
		{numCollectors: 100, numJobs: 10000},
		{numCollectors: 100, numJobs: 100000},
		{numCollectors: 1000, numJobs: 100},
		{numCollectors: 1000, numJobs: 1000},
		{numCollectors: 1000, numJobs: 10000},
		{numCollectors: 1000, numJobs: 100000},
	}

	for _, allocatorName := range allocation.GetRegisteredAllocatorNames() {
		for _, v := range table {
			a, _ := allocation.New(allocatorName, logger)
			cols := allocation.MakeNCollectors(v.numCollectors, 0)
			targets := allocation.MakeNNewTargets(v.numJobs, v.numCollectors, 0)
			listenAddr := ":8080"
			a.SetCollectors(cols)
			a.SetTargets(targets)
			s := NewServer(logger, a, listenAddr)
			b.Run(fmt.Sprintf("%s_num_cols_%d_num_jobs_%d", allocatorName, v.numCollectors, v.numJobs), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					randomJob := random.Intn(v.numJobs)       //nolint: gosec
					randomCol := random.Intn(v.numCollectors) //nolint: gosec
					request := httptest.NewRequest("GET", fmt.Sprintf("/jobs/test-job-%d/targets?collector_id=collector-%d", randomJob, randomCol), nil)
					w := httptest.NewRecorder()
					s.server.Handler.ServeHTTP(w, request)
				}
			})
		}
	}
}

func BenchmarkScrapeConfigsHandler(b *testing.B) {
	random := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
	s := &Server{
		logger: logger,
	}

	tests := []int{0, 5, 10, 50, 100, 500}
	for _, n := range tests {
		data := makeNScrapeConfigs(*random, n)
		assert.NoError(b, s.UpdateScrapeConfigResponse(data))

		b.Run(fmt.Sprintf("%d_targets", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				gin.SetMode(gin.ReleaseMode)
				c.Request = httptest.NewRequest("GET", "/scrape_configs", nil)

				s.ScrapeConfigsHandler(c)
			}
		})
	}
}

func BenchmarkCollectorMapJSONHandler(b *testing.B) {
	random := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
	s := &Server{
		logger: logger,
	}

	tests := []struct {
		numCollectors int
		numTargets    int
	}{
		{
			numCollectors: 0,
			numTargets:    0,
		},
		{
			numCollectors: 5,
			numTargets:    5,
		},
		{
			numCollectors: 5,
			numTargets:    50,
		},
		{
			numCollectors: 5,
			numTargets:    500,
		},
		{
			numCollectors: 50,
			numTargets:    5,
		},
		{
			numCollectors: 50,
			numTargets:    50,
		},
		{
			numCollectors: 50,
			numTargets:    500,
		},
		{
			numCollectors: 50,
			numTargets:    5000,
		},
	}
	for _, tc := range tests {
		data := makeNCollectorJSON(*random, tc.numCollectors, tc.numTargets)
		b.Run(fmt.Sprintf("%d_collectors_%d_targets", tc.numCollectors, tc.numTargets), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				resp := httptest.NewRecorder()
				s.jsonHandler(resp, data)
			}
		})
	}
}

func BenchmarkTargetItemsJSONHandler(b *testing.B) {
	random := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
	s := &Server{
		logger: logger,
	}

	tests := []struct {
		numTargets int
		numLabels  int
	}{
		{
			numTargets: 0,
			numLabels:  0,
		},
		{
			numTargets: 5,
			numLabels:  5,
		},
		{
			numTargets: 5,
			numLabels:  50,
		},
		{
			numTargets: 50,
			numLabels:  5,
		},
		{
			numTargets: 50,
			numLabels:  50,
		},
		{
			numTargets: 500,
			numLabels:  50,
		},
		{
			numTargets: 500,
			numLabels:  500,
		},
		{
			numTargets: 5000,
			numLabels:  50,
		},
		{
			numTargets: 5000,
			numLabels:  500,
		},
	}
	for _, tc := range tests {
		data := makeNTargetJSON(*random, tc.numTargets, tc.numLabels)
		b.Run(fmt.Sprintf("%d_targets_%d_labels", tc.numTargets, tc.numLabels), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				resp := httptest.NewRecorder()
				s.jsonHandler(resp, data)
			}
		})
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_/")

func randSeq(random rand.Rand, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[random.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}

func makeNScrapeConfigs(random rand.Rand, n int) map[string]*promconfig.ScrapeConfig {
	items := make(map[string]*promconfig.ScrapeConfig, n)
	for i := 0; i < n; i++ {
		items[randSeq(random, 20)] = &promconfig.ScrapeConfig{
			JobName:               randSeq(random, 20),
			ScrapeInterval:        model.Duration(30 * time.Second),
			ScrapeTimeout:         model.Duration(time.Minute),
			MetricsPath:           randSeq(random, 50),
			SampleLimit:           5,
			TargetLimit:           200,
			LabelLimit:            20,
			LabelNameLengthLimit:  50,
			LabelValueLengthLimit: 100,
		}
	}
	return items
}

func makeNCollectorJSON(random rand.Rand, numCollectors, numItems int) map[string]collectorJSON {
	items := make(map[string]collectorJSON, numCollectors)
	for i := 0; i < numCollectors; i++ {
		items[randSeq(random, 20)] = collectorJSON{
			Link: randSeq(random, 120),
			Jobs: makeNTargetJSON(random, numItems, 50),
		}
	}
	return items
}

func makeNTargetItems(random rand.Rand, numItems, numLabels int) []*target.Item {
	builder := labels.NewBuilder(labels.EmptyLabels())
	items := make([]*target.Item, 0, numItems)
	for i := 0; i < numItems; i++ {
		items = append(items, target.NewItem(
			randSeq(random, 80),
			randSeq(random, 150),
			makeNNewLabels(builder, random, numLabels),
			randSeq(random, 30),
		))
	}
	return items
}

func makeNTargetJSON(random rand.Rand, numItems, numLabels int) []*targetJSON {
	items := makeNTargetItems(random, numItems, numLabels)
	targets := make([]*targetJSON, numItems)
	for i := 0; i < numItems; i++ {
		targets[i] = targetJsonFromTargetItem(items[i])
	}
	return targets
}

func makeNNewLabels(builder *labels.Builder, random rand.Rand, n int) labels.Labels {
	builder.Reset(labels.EmptyLabels())
	for i := 0; i < n; i++ {
		builder.Set(randSeq(random, 20), randSeq(random, 20))
	}
	return builder.Labels()
}
