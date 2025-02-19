// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

func getKubeConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(kubeConfigPathFlagName)
}

func getListenAddr(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(listenAddrFlagName)
}

func getPrometheusCREnabled(flagSet *pflag.FlagSet) (value bool, changed bool, err error) {
	if changed = flagSet.Changed(prometheusCREnabledFlagName); !changed {
		value, err = false, nil
		return
	}
	value, err = flagSet.GetBool(prometheusCREnabledFlagName)
	return
}

func getHttpsListenAddr(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	if changed = flagSet.Changed(listenAddrHttpsFlagName); !changed {
		value, err = ":8443", nil
		return
	}
	value, err = flagSet.GetString(listenAddrHttpsFlagName)
	return
}

func getHttpsEnabled(flagSet *pflag.FlagSet) (value bool, changed bool, err error) {
	if changed = flagSet.Changed(httpsEnabledFlagName); !changed {
		value, err = false, nil
		return
	}
	value, err = flagSet.GetBool(httpsEnabledFlagName)
	return
}

func getHttpsCAFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	if changed = flagSet.Changed(httpsCAFilePathFlagName); !changed {
		value, err = "", nil
		return
	}
	value, err = flagSet.GetString(httpsCAFilePathFlagName)
	return
}

func getHttpsTLSCertFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	if changed = flagSet.Changed(httpsTLSCertFilePathFlagName); !changed {
		value, err = "", nil
		return
	}
	value, err = flagSet.GetString(httpsTLSCertFilePathFlagName)
	return
}

func getHttpsTLSKeyFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	if changed = flagSet.Changed(httpsTLSKeyFilePathFlagName); !changed {
		value, err = "", nil
		return
	}
	value, err = flagSet.GetString(httpsTLSKeyFilePathFlagName)
	return
}
