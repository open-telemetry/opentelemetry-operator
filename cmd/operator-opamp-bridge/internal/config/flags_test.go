// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestGetFlagSet(t *testing.T) {
	fs := GetFlagSet(pflag.ExitOnError)

	// Check if each flag exists
	assert.NotNil(t, fs.Lookup(configFilePathFlagName), "Flag %s not found", configFilePathFlagName)
	assert.NotNil(t, fs.Lookup(listenAddrFlagName), "Flag %s not found", listenAddrFlagName)
	assert.NotNil(t, fs.Lookup(kubeConfigPathFlagName), "Flag %s not found", kubeConfigPathFlagName)
}

func TestFlagGetters(t *testing.T) {
	tests := []struct {
		name          string
		flagArgs      []string
		expectedValue interface{}
		expectedErr   bool
		getterFunc    func(*pflag.FlagSet) (interface{}, error)
	}{
		{
			name:          "GetConfigFilePath",
			flagArgs:      []string{"--" + configFilePathFlagName, "/path/to/config"},
			expectedValue: "/path/to/config",
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getConfigFilePath(fs) },
		},
		{
			name:          "GetKubeConfigFilePath",
			flagArgs:      []string{"--" + kubeConfigPathFlagName, filepath.Join("~", ".kube", "config")},
			expectedValue: filepath.Join("~", ".kube", "config"),
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getKubeConfigFilePath(fs) },
		},
		{
			name:          "GetName",
			flagArgs:      []string{"--" + nameFlagName, "test"},
			expectedValue: "test",
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getName(fs) },
		},
		{
			name:          "GetListenAddr",
			flagArgs:      []string{"--" + listenAddrFlagName, ":8081"},
			expectedValue: ":8081",
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getListenAddr(fs) },
		},
		{
			name:          "GetHeartbeatInterval",
			flagArgs:      []string{"--" + heartbeatIntervalFlagName, "45s"},
			expectedValue: 45 * time.Second,
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getHeartbeatInterval(fs) },
		},
		{
			name:        "InvalidFlag",
			flagArgs:    []string{"--invalid-flag", "value"},
			expectedErr: true,
			getterFunc:  func(fs *pflag.FlagSet) (interface{}, error) { return getConfigFilePath(fs) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := GetFlagSet(pflag.ContinueOnError)
			err := fs.Parse(tt.flagArgs)

			// If an error is expected during parsing, we check it here.
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			got, err := tt.getterFunc(fs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, got)
		})
	}
}
