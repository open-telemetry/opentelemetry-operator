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
	"flag"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Flag names.
const (
	targetAllocatorName          = "target-allocator"
	configFilePathFlagName       = "config-file"
	listenAddrFlagName           = "listen-addr"
	prometheusCREnabledFlagName  = "enable-prometheus-cr-watcher"
	kubeConfigPathFlagName       = "kubeconfig-path"
	httpsEnabledFlagName         = "enable-https-server"
	listenAddrHttpsFlagName      = "listen-addr-https"
	httpsCAFilePathFlagName      = "https-ca-file"
	httpsTLSCertFilePathFlagName = "https-tls-cert-file"
	httpsTLSKeyFilePathFlagName  = "https-tls-key-file"
)

// We can't bind this flag to our FlagSet, so we need to handle it separately.
var zapCmdLineOpts zap.Options

func getFlagSet(errorHandling pflag.ErrorHandling) *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(targetAllocatorName, errorHandling)
	flagSet.String(configFilePathFlagName, DefaultConfigFilePath, "The path to the config file.")
	flagSet.String(listenAddrFlagName, ":8080", "The address where this service serves.")
	flagSet.Bool(prometheusCREnabledFlagName, false, "Enable Prometheus CRs as target sources")
	flagSet.String(kubeConfigPathFlagName, filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file")
	flagSet.Bool(httpsEnabledFlagName, false, "Enable HTTPS additional server")
	flagSet.String(listenAddrHttpsFlagName, ":8443", "The address where this service serves over HTTPS.")
	flagSet.String(httpsCAFilePathFlagName, DefaultHttpsCAFilePath, "The path to the HTTPS server TLS CA file.")
	flagSet.String(httpsTLSCertFilePathFlagName, DefaultHttpsTLSCertFilePath, "The path to the HTTPS server TLS certificate file.")
	flagSet.String(httpsTLSKeyFilePathFlagName, DefaultHttpsTLSKeyFilePath, "The path to the HTTPS server TLS key file.")
	zapFlagSet := flag.NewFlagSet("", flag.ErrorHandling(errorHandling))
	zapCmdLineOpts.BindFlags(zapFlagSet)
	flagSet.AddGoFlagSet(zapFlagSet)
	return flagSet
}

func getConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(configFilePathFlagName)
}

func getKubeConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(kubeConfigPathFlagName)
}

func getListenAddr(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(listenAddrFlagName)
}

func getPrometheusCREnabled(flagSet *pflag.FlagSet) (bool, error) {
	return flagSet.GetBool(prometheusCREnabledFlagName)
}

func getHttpsListenAddr(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(listenAddrHttpsFlagName)
}

func getHttpsEnabled(flagSet *pflag.FlagSet) (bool, error) {
	return flagSet.GetBool(httpsEnabledFlagName)
}

func getHttpsCAFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(httpsCAFilePathFlagName)
}

func getHttpsTLSCertFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(httpsTLSCertFilePathFlagName)
}

func getHttpsTLSKeyFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(httpsTLSKeyFilePathFlagName)
}
