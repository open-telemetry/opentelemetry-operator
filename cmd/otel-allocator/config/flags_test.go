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

package config

import (
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestGetFlagSet(t *testing.T) {
	fs := getFlagSet(pflag.ExitOnError)

	// Check if each flag exists
	assert.NotNil(t, fs.Lookup(configFilePathFlagName), "Flag %s not found", configFilePathFlagName)
	assert.NotNil(t, fs.Lookup(listenAddrFlagName), "Flag %s not found", listenAddrFlagName)
	assert.NotNil(t, fs.Lookup(prometheusCREnabledFlagName), "Flag %s not found", prometheusCREnabledFlagName)
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
			name:          "GetListenAddr",
			flagArgs:      []string{"--" + listenAddrFlagName, ":8081"},
			expectedValue: ":8081",
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getListenAddr(fs) },
		},
		{
			name:          "GetPrometheusCREnabled",
			flagArgs:      []string{"--" + prometheusCREnabledFlagName, "true"},
			expectedValue: true,
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getPrometheusCREnabled(fs) },
		},
		{
			name:          "GetConfigReloadEnabled",
			flagArgs:      []string{"--" + reloadConfigFlagName, "true"},
			expectedValue: true,
			getterFunc:    func(fs *pflag.FlagSet) (interface{}, error) { return getConfigReloadEnabled(fs) },
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
			fs := getFlagSet(pflag.ContinueOnError)
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
