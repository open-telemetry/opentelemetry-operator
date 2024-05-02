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
		logger:         logger,
		jsonMarshaller: jsonConfig,
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
		logger:         logger,
		jsonMarshaller: jsonConfig,
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
		data := makeNTargetItems(*random, tc.numTargets, tc.numLabels)
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
			Jobs: makeNTargetItems(random, numItems, 50),
		}
	}
	return items
}

func makeNTargetItems(random rand.Rand, numItems, numLabels int) []*target.Item {
	items := make([]*target.Item, 0, numItems)
	for i := 0; i < numItems; i++ {
		items = append(items, target.NewItem(
			randSeq(random, 80),
			randSeq(random, 150),
			makeNNewLabels(random, numLabels),
			randSeq(random, 30),
		))
	}
	return items
}

func makeNNewLabels(random rand.Rand, n int) model.LabelSet {
	labels := make(map[model.LabelName]model.LabelValue, n)
	for i := 0; i < n; i++ {
		labels[model.LabelName(randSeq(random, 20))] = model.LabelValue(randSeq(random, 20))
	}
	return labels
}
