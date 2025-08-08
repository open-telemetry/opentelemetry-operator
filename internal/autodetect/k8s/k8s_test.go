// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"testing"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi"
	restclient "k8s.io/client-go/rest"
)

type mockDiscoveryClient struct {
	mock.Mock
}

func (m *mockDiscoveryClient) ServerVersion() (*version.Info, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*version.Info), args.Error(1)
}

func (m *mockDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	args := m.Called()
	return args.Get(0).(*openapi_v2.Document), args.Error(1)
}

func (m *mockDiscoveryClient) OpenAPIV3() openapi.Client {
	args := m.Called()
	return args.Get(0).(openapi.Client)
}

func (m *mockDiscoveryClient) RESTClient() restclient.Interface {
	args := m.Called()
	return args.Get(0).(restclient.Interface)
}

func (m *mockDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	args := m.Called()
	return args.Get(0).(*metav1.APIGroupList), args.Error(1)
}

func (m *mockDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	args := m.Called()
	return args.Get(0).([]*metav1.APIGroup), args.Get(1).([]*metav1.APIResourceList), args.Error(2)
}

func (m *mockDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	args := m.Called()
	return args.Get(0).([]*metav1.APIResourceList), args.Error(1)
}

func (m *mockDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	args := m.Called()
	return args.Get(0).([]*metav1.APIResourceList), args.Error(1)
}

func (m *mockDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	args := m.Called(groupVersion)
	return args.Get(0).(*metav1.APIResourceList), args.Error(1)
}

func (m *mockDiscoveryClient) WithLegacy() discovery.DiscoveryInterface {
	args := m.Called()
	return args.Get(0).(discovery.DiscoveryInterface)
}

func TestGetKubernetesVersion(t *testing.T) {
	tests := []struct {
		name          string
		gitVersion    string
		serverError   error
		expectedError string
		expectedMajor uint
		expectedMinor uint
	}{
		{
			name:          "successful version parsing - standard version",
			gitVersion:    "v1.28.5",
			expectedMajor: 1,
			expectedMinor: 28,
		},
		{
			name:          "successful version parsing - with patch and build info",
			gitVersion:    "v1.29.2+k3s1",
			expectedMajor: 1,
			expectedMinor: 29,
		},
		{
			name:          "successful version parsing - with pre-release",
			gitVersion:    "v1.30.0-alpha.1",
			expectedMajor: 1,
			expectedMinor: 30,
		},
		{
			name:          "server version error",
			serverError:   errors.New("connection refused"),
			expectedError: "failed to get server version: connection refused",
		},
		{
			name:          "invalid version format",
			gitVersion:    "invalid-version",
			expectedError: "failed to parse server version \"invalid-version\":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockDiscoveryClient{}
			detector := NewDetector(mockClient)

			if tt.serverError != nil {
				mockClient.On("ServerVersion").Return(nil, tt.serverError)
			} else {
				versionInfo := &version.Info{
					GitVersion: tt.gitVersion,
				}
				mockClient.On("ServerVersion").Return(versionInfo, nil)
			}

			result, err := detector.GetKubernetesVersion()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedMajor, result.Major())
				assert.Equal(t, tt.expectedMinor, result.Minor())
			}

			mockClient.AssertExpectations(t)
		})
	}
}
