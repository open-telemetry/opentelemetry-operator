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

package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"
)

var (
	l                    = logr.Discard()
	_ client.OpAMPClient = &mockOpampClient{}
)

type mockOpampClient struct {
	lastStatus          *protobufs.RemoteConfigStatus
	lastEffectiveConfig *protobufs.EffectiveConfig
	settings            types.StartSettings
}

func (m *mockOpampClient) Start(ctx context.Context, settings types.StartSettings) error {
	m.settings = settings
	return nil
}

func (m *mockOpampClient) Stop(ctx context.Context) error {
	return nil
}

func (m *mockOpampClient) SetAgentDescription(descr *protobufs.AgentDescription) error {
	return nil
}

func (m *mockOpampClient) AgentDescription() *protobufs.AgentDescription {
	return nil
}

func (m *mockOpampClient) SetHealth(health *protobufs.AgentHealth) error {
	return nil
}

func (m *mockOpampClient) UpdateEffectiveConfig(ctx context.Context) error {
	effectiveConfig, err := m.settings.Callbacks.GetEffectiveConfig(ctx)
	if err != nil {
		return err
	}
	m.lastEffectiveConfig = effectiveConfig
	return nil
}

func (m *mockOpampClient) SetRemoteConfigStatus(status *protobufs.RemoteConfigStatus) error {
	m.lastStatus = status
	return nil
}

func (m *mockOpampClient) SetPackageStatuses(statuses *protobufs.PackageStatuses) error {
	return nil
}

func getFakeApplier(t *testing.T, conf *config.Config) *operator.Client {
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
		metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err, "Should be able to add custom types")
	c := fake.NewClientBuilder().WithScheme(scheme)
	return operator.NewClient(l, c.Build(), conf.GetComponentsAllowed())
}

func TestAgent_onMessage(t *testing.T) {
	type fields struct {
		configFile string
	}
	type args struct {
		ctx context.Context
		// Mapping from namespace/name to a config in testdata
		configFile map[string]string
		// Mapping from namespace/name to a config in testdata (for testing updates)
		nextConfigFile map[string]string
	}
	type want struct {
		// Mapping from namespace/name to a list of expected contents
		contents map[string][]string
		// Mapping from namespace/name to a list of updated expected contents
		nextContents map[string][]string
		// The status after the initial config loading
		status *protobufs.RemoteConfigStatus
		// The status after the updated config loading
		nextStatus *protobufs.RemoteConfigStatus
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "no data",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx:        context.Background(),
				configFile: nil,
			},
			want: want{
				contents: map[string][]string{},
				status:   nil,
			},
		},
		{
			name: "base case",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"receivers: [otlp]",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "base case http",
			fields: fields{
				configFile: "testdata/agenthttpbasic.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"receivers: [otlp]",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "failure",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/bad": "invalid.yaml",
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/bad513"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "error converting YAML to JSON: yaml: line 21: could not find expected ':'",
				},
			},
		},
		{
			name: "all components are allowed",
			fields: fields{
				configFile: "testdata/agentbasiccomponentsallowed.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"receivers: [otlp]",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "batch not allowed",
			fields: fields{
				configFile: "testdata/agentbatchnotallowed.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "Items in config are not allowed: [processors.batch]",
				},
			},
		},
		{
			name: "processors not allowed",
			fields: fields{
				configFile: "testdata/agentnoprocessorsallowed.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "Items in config are not allowed: [processors]",
				},
			},
		},
		{
			name: "can update config and replicas",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
				nextConfigFile: map[string]string{
					"testnamespace/good": "updated.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"replicas: 1",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: [memory_limiter, batch]",
						"replicas: 3",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good547"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "cannot update with bad config",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
				nextConfigFile: map[string]string{
					"testnamespace/good": "invalid.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"replicas: 1",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"replicas: 1",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good513"), // The new hash should be of the bad config
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "error converting YAML to JSON: yaml: line 21: could not find expected ':'",
				},
			},
		},
		{
			name: "update with new collector",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
				nextConfigFile: map[string]string{
					"testnamespace/good":  "basic.yaml",
					"testnamespace/other": "updated.yaml",
				},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"status:",
					},
					"testnamespace/other": {
						"kind: OpenTelemetryCollector",
						"name: other",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: [memory_limiter, batch]",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512testnamespace/other547"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "can delete existing collector",
			fields: fields{
				configFile: "testdata/agent.yaml",
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					"testnamespace/good": "basic.yaml",
				},
				nextConfigFile: map[string]string{},
			},
			want: want{
				contents: map[string][]string{
					"testnamespace/good": {
						"kind: OpenTelemetryCollector",
						"name: good",
						"namespace: testnamespace",
						"send_batch_size: 10000",
						"processors: []",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte("testnamespace/good512"),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(""),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockOpampClient{}
			conf := config.NewConfig(logr.Discard())
			loadErr := config.LoadFromFile(conf, tt.fields.configFile)
			require.NoError(t, loadErr, "should be able to load config")
			applier := getFakeApplier(t, conf)
			agent := NewAgent(l, applier, conf, mockClient)
			err := agent.Start()
			defer agent.Shutdown()
			require.NoError(t, err, "should be able to start agent")
			data, err := getMessageDataFromConfigFile(tt.args.configFile)
			require.NoError(t, err, "should be able to load data")
			agent.onMessage(tt.args.ctx, data)
			effectiveConfig, err := agent.getEffectiveConfig(tt.args.ctx)
			require.NoError(t, err, "should be able to get effective config")
			if tt.args.configFile != nil {
				// We should only expect this to happen if we supply configuration
				assert.Equal(t, effectiveConfig, mockClient.lastEffectiveConfig, "client's config should be updated")
			}
			assert.NotNilf(t, effectiveConfig.ConfigMap.GetConfigMap(), "configmap should have data")
			for colNameNamespace, expectedContents := range tt.want.contents {
				assert.Contains(t, effectiveConfig.ConfigMap.GetConfigMap(), colNameNamespace)
				for _, content := range expectedContents {
					asString := string(effectiveConfig.ConfigMap.GetConfigMap()[colNameNamespace].GetBody())
					assert.Contains(t, asString, content)
				}
			}
			assert.Equal(t, tt.want.status, mockClient.lastStatus)
			if tt.args.nextConfigFile == nil {
				// Nothing left to do!
				return
			}
			nextData, err := getMessageDataFromConfigFile(tt.args.nextConfigFile)
			require.NoError(t, err, "should be able to load updated data")
			agent.onMessage(tt.args.ctx, nextData)
			nextEffectiveConfig, err := agent.getEffectiveConfig(tt.args.ctx)
			require.NoError(t, err, "should be able to get updated effective config")
			assert.Equal(t, nextEffectiveConfig, mockClient.lastEffectiveConfig, "client's config should be updated")
			assert.NotNilf(t, nextEffectiveConfig.ConfigMap.GetConfigMap(), "configmap should have updated data")
			for colNameNamespace, expectedContents := range tt.want.nextContents {
				assert.Contains(t, nextEffectiveConfig.ConfigMap.GetConfigMap(), colNameNamespace)
				for _, content := range expectedContents {
					asString := string(nextEffectiveConfig.ConfigMap.GetConfigMap()[colNameNamespace].GetBody())
					assert.Contains(t, asString, content)
				}
			}
			assert.Equal(t, tt.want.nextStatus, mockClient.lastStatus)
		})
	}
}

func Test_CanUpdateIdentity(t *testing.T) {
	mockClient := &mockOpampClient{}

	fs := config.GetFlagSet(pflag.ContinueOnError)
	configFlag := []string{"--config-file", "testdata/agent.yaml"}
	err := fs.Parse(configFlag)
	assert.NoError(t, err)
	conf := config.NewConfig(logr.Discard())
	loadErr := config.LoadFromFile(conf, "testdata/agent.yaml")
	require.NoError(t, loadErr, "should be able to load config")
	applier := getFakeApplier(t, conf)
	agent := NewAgent(l, applier, conf, mockClient)
	err = agent.Start()
	defer agent.Shutdown()
	require.NoError(t, err, "should be able to start agent")
	previousInstanceId := agent.instanceId.String()
	entropy := ulid.Monotonic(rand.Reader, 0)
	newId := ulid.MustNew(ulid.MaxTime(), entropy)
	agent.onMessage(context.Background(), &types.MessageData{
		AgentIdentification: &protobufs.AgentIdentification{
			NewInstanceUid: newId.String(),
		},
	})
	assert.NotEqual(t, previousInstanceId, newId.String())
	assert.Equal(t, agent.instanceId, newId)
}

func getMessageDataFromConfigFile(filemap map[string]string) (*types.MessageData, error) {
	toReturn := &types.MessageData{}
	if filemap == nil {
		return toReturn, nil
	}
	configs := map[string]*protobufs.AgentConfigFile{}
	hash := ""
	fileNames := make([]string, len(filemap))
	i := 0
	for k := range filemap {
		fileNames[i] = k
		i++
	}
	// We sort the filenames so we get consistent results for multiple file loads
	sort.Strings(fileNames)

	for _, key := range fileNames {
		yamlFile, err := os.ReadFile(fmt.Sprintf("testdata/%s", filemap[key]))
		if err != nil {
			return toReturn, err
		}
		configs[key] = &protobufs.AgentConfigFile{
			Body:        yamlFile,
			ContentType: "yaml",
		}
		hash = hash + key + fmt.Sprint(len(yamlFile))
	}
	toReturn.RemoteConfig = &protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: configs,
		},
		// just use the file name for the hash
		ConfigHash: []byte(hash),
	}
	return toReturn, nil
}
