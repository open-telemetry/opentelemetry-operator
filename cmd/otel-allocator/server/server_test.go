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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"io"
	"math/big"
	"net/http/httptest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	logger       = logf.Log.WithName("server-unit-tests")
	baseLabelSet = model.LabelSet{
		"test_label": "test-value",
	}
	testJobLabelSetTwo = model.LabelSet{
		"test_label": "test-value2",
	}
	baseTargetItem       = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	secondTargetItem     = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	testJobTargetItemTwo = target.NewItem("test-job", "test-url2", testJobLabelSetTwo, "test-collector2")
)

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func makeNNewTargets(n int, numCollectors int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := model.LabelSet{
			"collector": model.LabelValue(collector),
			"i":         model.LabelValue(strconv.Itoa(i)),
			"total":     model.LabelValue(strconv.Itoa(n + startingIndex)),
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), "test-url", label, collector)
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func makeNCollectors(n int, startingIndex int) map[string]*allocation.Collector {
	toReturn := map[string]*allocation.Collector{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", i)
		toReturn[collector] = &allocation.Collector{
			Name:       collector,
			NumTargets: 0,
		}
	}
	return toReturn
}

func TestServer_TargetsHandler(t *testing.T) {
	leastWeighted, _ := allocation.New("least-weighted", logger)
	type args struct {
		collector string
		job       string
		cMap      map[string]*target.Item
		allocator allocation.Allocator
	}
	type want struct {
		items     []*target.Item
		errString string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Empty target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap:      map[string]*target.Item{},
				allocator: leastWeighted,
			},
			want: want{
				items: []*target.Item{},
			},
		},
		{
			name: "Single entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string]*target.Item{
					baseTargetItem.Hash(): baseTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*target.Item{
					{
						TargetURL: []string{"test-url"},
						Labels: map[model.LabelName]model.LabelValue{
							"test_label": "test-value",
						},
					},
				},
			},
		},
		{
			name: "Multiple entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string]*target.Item{
					baseTargetItem.Hash():   baseTargetItem,
					secondTargetItem.Hash(): secondTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*target.Item{
					{
						TargetURL: []string{"test-url"},
						Labels: map[model.LabelName]model.LabelValue{
							"test_label": "test-value",
						},
					},
				},
			},
		},
		{
			name: "Multiple entry target map of same job with label merge",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string]*target.Item{
					baseTargetItem.Hash():       baseTargetItem,
					testJobTargetItemTwo.Hash(): testJobTargetItemTwo,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*target.Item{
					{
						TargetURL: []string{"test-url"},
						Labels: map[model.LabelName]model.LabelValue{
							"test_label": "test-value",
						},
					},
					{
						TargetURL: []string{"test-url2"},
						Labels: map[model.LabelName]model.LabelValue{
							"test_label": "test-value2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listenAddr := ":8080"
			s := NewServer(logger, tt.args.allocator, nil, &listenAddr)
			tt.args.allocator.SetCollectors(map[string]*allocation.Collector{"test-collector": {Name: "test-collector"}})
			tt.args.allocator.SetTargets(tt.args.cMap)
			request := httptest.NewRequest("GET", fmt.Sprintf("/jobs/%s/targets?collector_id=%s", tt.args.job, tt.args.collector), nil)
			w := httptest.NewRecorder()
			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			if len(tt.want.errString) != 0 {
				assert.EqualError(t, err, tt.want.errString)
				return
			}
			var itemResponse []*target.Item
			err = json.Unmarshal(bodyBytes, &itemResponse)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.items, itemResponse)
		})
	}
}

func randInt(max int64) int64 {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(max))
	return nBig.Int64()
}

func BenchmarkServerTargetsHandler(b *testing.B) {
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
			cols := makeNCollectors(v.numCollectors, 0)
			targets := makeNNewTargets(v.numJobs, v.numCollectors, 0)
			listenAddr := ":8080"
			a.SetCollectors(cols)
			a.SetTargets(targets)
			s := NewServer(logger, a, nil, &listenAddr)
			b.Run(fmt.Sprintf("%s_num_cols_%d_num_jobs_%d", allocatorName, v.numCollectors, v.numJobs), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					randomJob := randInt(int64(v.numJobs))
					randomCol := randInt(int64(v.numCollectors))
					request := httptest.NewRequest("GET", fmt.Sprintf("/jobs/test-job-%d/targets?collector_id=collector-%d", randomJob, randomCol), nil)
					w := httptest.NewRecorder()
					s.server.Handler.ServeHTTP(w, request)
				}
			})
		}
	}
}
