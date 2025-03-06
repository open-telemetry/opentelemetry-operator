// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testingclock "k8s.io/utils/clock/testing"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"
)

const (
	collectorBasicFile   = "testdata/basic.yaml"
	collectorUpdatedFile = "testdata/updated.yaml"
	collectorInvalidFile = "testdata/invalid.yaml"

	testNamespace      = "testnamespace"
	testCollectorName  = "collector"
	otherCollectorName = "other"
	thirdCollectorName = "third"
	emptyConfigHash    = ""
	testCollectorKey   = testNamespace + "/" + testCollectorName
	otherCollectorKey  = testNamespace + "/" + otherCollectorName
	thirdCollectorKey  = otherCollectorName + "/" + thirdCollectorName

	agentTestFileName                       = "testdata/agent.yaml"
	agentTestFileHttpName                   = "testdata/agenthttpbasic.yaml"
	agentTestFileBasicComponentsAllowedName = "testdata/agentbasiccomponentsallowed.yaml"
	agentTestFileBatchNotAllowedName        = "testdata/agentbatchnotallowed.yaml"
	agentTestFileNoProcessorsAllowedName    = "testdata/agentnoprocessorsallowed.yaml"

	collectorStartTime = uint64(0)
)

var (
	l                    = logr.Discard()
	_ client.OpAMPClient = &mockOpampClient{}

	basicYamlConfigHash        = getConfigHash(testCollectorKey, collectorBasicFile)
	invalidYamlConfigHash      = getConfigHash(testCollectorKey, collectorInvalidFile)
	updatedYamlConfigHash      = getConfigHash(testCollectorKey, collectorUpdatedFile)
	otherUpdatedYamlConfigHash = getConfigHash(otherCollectorKey, collectorUpdatedFile)

	podTime            = metav1.NewTime(time.Unix(0, 0))
	podTimeUnsigned, _ = timeToUnixNanoUnsigned(podTime.Time)
	mockPodList        = &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      thirdCollectorName + "-1",
					Namespace: otherCollectorName,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
						"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otherCollectorName, thirdCollectorName),
						"app.kubernetes.io/part-of":    "opentelemetry",
						"app.kubernetes.io/component":  "opentelemetry-collector",
					},
					CreationTimestamp: podTime,
				},
				Spec: v1.PodSpec{},
				Status: v1.PodStatus{
					StartTime: &podTime,
					Phase:     v1.PodRunning,
				},
			},
		}}
	mockPodListUnhealthy = &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      thirdCollectorName + "-1",
					Namespace: otherCollectorName,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
						"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otherCollectorName, thirdCollectorName),
						"app.kubernetes.io/part-of":    "opentelemetry",
						"app.kubernetes.io/component":  "opentelemetry-collector",
					},
					CreationTimestamp: podTime,
				},
				Spec: v1.PodSpec{},
				Status: v1.PodStatus{
					StartTime: nil,
					Phase:     v1.PodRunning,
				},
			},
		}}
)

func getConfigHash(key, file string) string {
	fi, err := os.Stat(file)
	if err != nil {
		return ""
	}
	// get the size
	size := fi.Size()
	return fmt.Sprintf("%s%d", key, size)
}

var _ client.OpAMPClient = &mockOpampClient{}

type mockOpampClient struct {
	lastStatus          *protobufs.RemoteConfigStatus
	lastEffectiveConfig *protobufs.EffectiveConfig
	settings            types.StartSettings
}

func (m *mockOpampClient) SetCustomCapabilities(_ *protobufs.CustomCapabilities) error {
	return nil
}

func (m *mockOpampClient) SendCustomMessage(_ *protobufs.CustomMessage) (messageSendingChannel chan struct{}, err error) {
	return nil, nil
}

func (m *mockOpampClient) RequestConnectionSettings(_ *protobufs.ConnectionSettingsRequest) error {
	return nil
}

func (m *mockOpampClient) Start(_ context.Context, settings types.StartSettings) error {
	m.settings = settings
	return nil
}

func (m *mockOpampClient) Stop(_ context.Context) error {
	return nil
}

func (m *mockOpampClient) SetAgentDescription(_ *protobufs.AgentDescription) error {
	return nil
}

func (m *mockOpampClient) AgentDescription() *protobufs.AgentDescription {
	return nil
}

func (m *mockOpampClient) SetHealth(_ *protobufs.ComponentHealth) error {
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

func (m *mockOpampClient) SetPackageStatuses(_ *protobufs.PackageStatuses) error {
	return nil
}

func getFakeApplier(t *testing.T, conf *config.Config, lists ...runtimeClient.ObjectList) *operator.Client {
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
		s.AddKnownTypes(v1beta1.GroupVersion, &v1beta1.OpenTelemetryCollector{}, &v1beta1.OpenTelemetryCollectorList{})
		s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})
		metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err, "Should be able to add custom types")
	c := fake.NewClientBuilder().WithLists(lists...).WithScheme(scheme)
	return operator.NewClient("test-bridge", l, c.Build(), conf.GetComponentsAllowed())
}

func TestAgent_getHealth(t *testing.T) {
	fakeClock := testingclock.NewFakeClock(time.Now())
	startTime, err := timeToUnixNanoUnsigned(fakeClock.Now())
	require.NoError(t, err)
	type fields struct {
		configFile string
	}
	type args struct {
		ctx context.Context
		// List of mappings from namespace/name to a config file, tests are run in order of list
		configs []map[string]string
		podList *v1.PodList
	}
	tests := []struct {
		name   string
		args   args
		fields fields
		// want is evaluated with the corresponding configs' index.
		want []*protobufs.ComponentHealth
	}{
		{
			name: "no data",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx:     context.Background(),
				configs: nil,
				podList: mockPodList,
			},
			want: []*protobufs.ComponentHealth{
				{
					Healthy:            true,
					StartTimeUnixNano:  startTime,
					LastError:          "",
					Status:             "",
					StatusTimeUnixNano: startTime,
					ComponentHealthMap: map[string]*protobufs.ComponentHealth{},
				},
			},
		},
		{
			name: "base case",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configs: []map[string]string{
					{
						testCollectorKey: collectorBasicFile,
					},
				},
				podList: mockPodList,
			},
			want: []*protobufs.ComponentHealth{
				{
					Healthy:            true,
					StartTimeUnixNano:  startTime,
					StatusTimeUnixNano: startTime,
					ComponentHealthMap: map[string]*protobufs.ComponentHealth{
						"testnamespace/collector": {
							Healthy:            true,
							StartTimeUnixNano:  collectorStartTime,
							LastError:          "",
							Status:             "",
							StatusTimeUnixNano: startTime,
							ComponentHealthMap: map[string]*protobufs.ComponentHealth{},
						},
					},
				},
			},
		},
		{
			name: "two collectors",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configs: []map[string]string{
					{
						testCollectorKey:  collectorBasicFile,
						otherCollectorKey: collectorUpdatedFile,
					},
				},
				podList: mockPodList,
			},
			want: []*protobufs.ComponentHealth{
				{
					Healthy:            true,
					StartTimeUnixNano:  startTime,
					StatusTimeUnixNano: startTime,
					ComponentHealthMap: map[string]*protobufs.ComponentHealth{
						"testnamespace/collector": {
							Healthy:            true,
							StartTimeUnixNano:  collectorStartTime,
							LastError:          "",
							Status:             "",
							StatusTimeUnixNano: startTime,
							ComponentHealthMap: map[string]*protobufs.ComponentHealth{},
						},
						"testnamespace/other": {
							Healthy:            true,
							StartTimeUnixNano:  collectorStartTime,
							LastError:          "",
							Status:             "",
							StatusTimeUnixNano: startTime,
							ComponentHealthMap: map[string]*protobufs.ComponentHealth{},
						},
					},
				},
			},
		},
		{
			name: "with pod health",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configs: []map[string]string{
					{
						thirdCollectorKey: collectorBasicFile,
					},
				},
				podList: mockPodList,
			},
			want: []*protobufs.ComponentHealth{
				{
					Healthy:            true,
					StartTimeUnixNano:  startTime,
					StatusTimeUnixNano: startTime,
					ComponentHealthMap: map[string]*protobufs.ComponentHealth{
						"other/third": {
							Healthy:            true,
							StartTimeUnixNano:  collectorStartTime,
							LastError:          "",
							Status:             "",
							StatusTimeUnixNano: startTime,
							ComponentHealthMap: map[string]*protobufs.ComponentHealth{
								otherCollectorName + "/" + thirdCollectorName + "-1": {
									Healthy:            true,
									Status:             "Running",
									StatusTimeUnixNano: startTime,
									StartTimeUnixNano:  podTimeUnsigned,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "with pod health, nil start time",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configs: []map[string]string{
					{
						thirdCollectorKey: collectorBasicFile,
					},
				},
				podList: mockPodListUnhealthy,
			},
			want: []*protobufs.ComponentHealth{
				{
					Healthy:            true,
					StartTimeUnixNano:  startTime,
					StatusTimeUnixNano: startTime,
					ComponentHealthMap: map[string]*protobufs.ComponentHealth{
						"other/third": {
							Healthy:            false, // we're working with mocks so the status will never be reconciled.
							StartTimeUnixNano:  collectorStartTime,
							LastError:          "",
							Status:             "",
							StatusTimeUnixNano: startTime,
							ComponentHealthMap: map[string]*protobufs.ComponentHealth{
								otherCollectorName + "/" + thirdCollectorName + "-1": {
									Healthy:            false,
									Status:             "Running",
									StatusTimeUnixNano: startTime,
									StartTimeUnixNano:  uint64(0),
								},
							},
						},
					},
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
			applier := getFakeApplier(t, conf, tt.args.podList)
			agent := NewAgent(l, applier, conf, mockClient)
			agent.clock = fakeClock
			err := agent.Start()
			defer agent.Shutdown()

			require.NoError(t, err, "should be able to start agent")
			if len(tt.args.configs) > 0 {
				require.Len(t, tt.args.configs, len(tt.want), "must have an equal amount of configs and checks.")
			} else {
				require.Len(t, tt.want, 1, "must have exactly one want if no config is supplied.")
				require.Equal(t, tt.want[0], agent.getHealth())
			}

			for i, configMap := range tt.args.configs {
				var data *types.MessageData
				data, err := getMessageDataFromConfigFile(configMap)
				require.NoError(t, err, "should be able to load data")
				agent.onMessage(tt.args.ctx, data)
				effectiveConfig, err := agent.getEffectiveConfig(tt.args.ctx)
				require.NoError(t, err, "should be able to get effective config")
				// We should only expect this to happen if we supply configuration
				assert.Equal(t, effectiveConfig, mockClient.lastEffectiveConfig, "client's config should be updated")
				assert.NotNilf(t, effectiveConfig.ConfigMap.GetConfigMap(), "configmap should have data")
				assert.Equal(t, tt.want[i], agent.getHealth())
			}
		})
	}
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
				configFile: agentTestFileName,
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
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"receivers:",
						"- otlp",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "base case http",
			fields: fields{
				configFile: agentTestFileHttpName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"grpc:",
						"receivers:",
						"- otlp",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "failure",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorInvalidFile,
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(invalidYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "failed to unmarshal config into v1beta1 API Version: error converting YAML to JSON: yaml: line 23: could not find expected ':'",
				},
			},
		},
		{
			name: "all components are allowed",
			fields: fields{
				configFile: agentTestFileBasicComponentsAllowedName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"receivers:",
						"- otlp",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "batch not allowed",
			fields: fields{
				configFile: agentTestFileBatchNotAllowedName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "Items in config are not allowed: [processors.batch]",
				},
			},
		},
		{
			name: "processors not allowed",
			fields: fields{
				configFile: agentTestFileNoProcessorsAllowedName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
			},
			want: want{
				contents: nil,
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "Items in config are not allowed: [processors]",
				},
			},
		},
		{
			name: "can update config and replicas",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
				nextConfigFile: map[string]string{
					testCollectorKey: collectorUpdatedFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"processors:",
						"- memory_limiter",
						"- batch",
						"replicas: 3",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(updatedYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "cannot update with bad config",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
				nextConfigFile: map[string]string{
					testCollectorKey: collectorInvalidFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(invalidYamlConfigHash), // The new hash should be of the bad config
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
					ErrorMessage:         "failed to unmarshal config into v1beta1 API Version: error converting YAML to JSON: yaml: line 23: could not find expected ':'",
				},
			},
		},
		{
			name: "update with new collector",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
				nextConfigFile: map[string]string{
					testCollectorKey:  collectorBasicFile,
					otherCollectorKey: collectorUpdatedFile,
				},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
					otherCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + otherCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"processors:",
						"- memory_limiter",
						"- batch",
						"status:",
					},
				},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash + otherUpdatedYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
			},
		},
		{
			name: "can delete existing collector",
			fields: fields{
				configFile: agentTestFileName,
			},
			args: args{
				ctx: context.Background(),
				configFile: map[string]string{
					testCollectorKey: collectorBasicFile,
				},
				nextConfigFile: map[string]string{},
			},
			want: want{
				contents: map[string][]string{
					testCollectorKey: {
						"kind: OpenTelemetryCollector",
						"name: " + testCollectorName,
						"namespace: " + testNamespace,
						"send_batch_size: 10000",
						"status:",
					},
				},
				status: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(basicYamlConfigHash),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				},
				nextContents: map[string][]string{},
				nextStatus: &protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: []byte(emptyConfigHash),
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
				configFileMap := effectiveConfig.ConfigMap.GetConfigMap()
				require.Contains(t, configFileMap, colNameNamespace)
				configFileString := string(configFileMap[colNameNamespace].GetBody())
				for _, content := range expectedContents {
					assert.Contains(t, configFileString, content, "config should contain %s", content)
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
				configFileMap := nextEffectiveConfig.ConfigMap.GetConfigMap()
				require.Contains(t, configFileMap, colNameNamespace)
				configFileString := string(configFileMap[colNameNamespace].GetBody())
				for _, content := range expectedContents {
					assert.Contains(t, configFileString, content)
				}
			}
			assert.Equal(t, tt.want.nextStatus, mockClient.lastStatus)
		})
	}
}

func Test_CanUpdateIdentity(t *testing.T) {
	mockClient := &mockOpampClient{}

	fs := config.GetFlagSet(pflag.ContinueOnError)
	configFlag := []string{"--config-file", agentTestFileName}
	err := fs.Parse(configFlag)
	assert.NoError(t, err)
	conf := config.NewConfig(logr.Discard())
	loadErr := config.LoadFromFile(conf, agentTestFileName)
	require.NoError(t, loadErr, "should be able to load config")
	applier := getFakeApplier(t, conf)
	agent := NewAgent(l, applier, conf, mockClient)
	err = agent.Start()
	defer agent.Shutdown()
	require.NoError(t, err, "should be able to start agent")
	previousInstanceId := agent.instanceId.String()
	newId, err := uuid.NewV7()
	require.NoError(t, err)
	marshalledId, err := newId.MarshalBinary()
	require.NoError(t, err)
	agent.onMessage(context.Background(), &types.MessageData{
		AgentIdentification: &protobufs.AgentIdentification{
			NewInstanceUid: marshalledId,
		},
	})
	assert.NotEqual(t, previousInstanceId, newId.String())
	assert.Equal(t, agent.instanceId, newId)
	parsedUUID, err := uuid.FromBytes(marshalledId)
	require.NoError(t, err)
	assert.Equal(t, newId, parsedUUID)
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
		yamlFile, err := os.ReadFile(filemap[key])
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
