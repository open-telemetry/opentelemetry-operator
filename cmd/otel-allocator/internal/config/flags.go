// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"

	"github.com/spf13/pflag"
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
	flagSet.String(listenAddrFlagName, DefaultListenAddr, "The address where this service serves.")
	flagSet.Bool(prometheusCREnabledFlagName, false, "Enable Prometheus CRs as target sources")
	flagSet.String(kubeConfigPathFlagName, DefaultKubeConfigFilePath, "absolute path to the KubeconfigPath file")
	flagSet.Bool(httpsEnabledFlagName, false, "Enable HTTPS additional server")
	flagSet.String(listenAddrHttpsFlagName, DefaultHttpsListenAddr, "The address where this service serves over HTTPS.")
	flagSet.String(httpsCAFilePathFlagName, "", "The path to the HTTPS server TLS CA file.")
	flagSet.String(httpsTLSCertFilePathFlagName, "", "The path to the HTTPS server TLS certificate file.")
	flagSet.String(httpsTLSKeyFilePathFlagName, "", "The path to the HTTPS server TLS key file.")
	zapFlagSet := flag.NewFlagSet("", flag.ErrorHandling(errorHandling))
	zapCmdLineOpts.BindFlags(zapFlagSet)
	flagSet.AddGoFlagSet(zapFlagSet)
	return flagSet
}

func getConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(configFilePathFlagName)
}

func getKubeConfigFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, kubeConfigPathFlagName)
}

func getListenAddr(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, listenAddrFlagName)
}

func getPrometheusCREnabled(flagSet *pflag.FlagSet) (value bool, changed bool, err error) {
	return getFlagValueAndChangedBool(flagSet, prometheusCREnabledFlagName)
}

func getHttpsListenAddr(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, listenAddrHttpsFlagName)
}

func getHttpsEnabled(flagSet *pflag.FlagSet) (value bool, changed bool, err error) {
	return getFlagValueAndChangedBool(flagSet, httpsEnabledFlagName)
}

func getHttpsCAFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, httpsCAFilePathFlagName)
}

func getHttpsTLSCertFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, httpsTLSCertFilePathFlagName)
}

func getHttpsTLSKeyFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChangedString(flagSet, httpsTLSKeyFilePathFlagName)
}

// getFlagValueAndChanged returns the given flag's string value and whether it was changed.
func getFlagValueAndChangedString(flagSet *pflag.FlagSet, flagName string) (value string, changed bool, err error) {
	if changed = flagSet.Changed(flagName); !changed {
		value, err = "", nil
		return
	}
	value, err = flagSet.GetString(flagName)
	return
}

// getFlagValueAndChanged returns the given flag's string value and whether it was changed.
func getFlagValueAndChangedBool(flagSet *pflag.FlagSet, flagName string) (value bool, changed bool, err error) {
	if changed = flagSet.Changed(flagName); !changed {
		value, err = false, nil
		return
	}
	value, err = flagSet.GetBool(flagName)
	return
}
