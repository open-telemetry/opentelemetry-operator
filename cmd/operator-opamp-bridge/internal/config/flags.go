// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Flag names.
const (
	opampBridgeName           = "opamp-bridge"
	defaultConfigFilePath     = "/conf/remoteconfiguration.yaml"
	configFilePathFlagName    = "config-file"
	listenAddrFlagName        = "listen-addr"
	defaultServerListenAddr   = ":8080"
	kubeConfigPathFlagName    = "kubeconfig-path"
	heartbeatIntervalFlagName = "heartbeat-interval"
	nameFlagName              = "name"
	defaultHeartbeatInterval  = 30 * time.Second
)

var (
	defaultKubeConfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
)

// We can't bind this flag to our FlagSet, so we need to handle it separately.
var zapCmdLineOpts zap.Options

func GetFlagSet(errorHandling pflag.ErrorHandling) *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(opampBridgeName, errorHandling)
	flagSet.String(configFilePathFlagName, defaultConfigFilePath, "The path to the config file.")
	flagSet.String(listenAddrFlagName, defaultServerListenAddr, "The address where this service serves.")
	flagSet.String(kubeConfigPathFlagName, defaultKubeConfigPath, "absolute path to the KubeconfigPath file.")
	flagSet.Duration(heartbeatIntervalFlagName, defaultHeartbeatInterval, "The interval to use for sending a heartbeat. Setting it to 0 disables the heartbeat.")
	flagSet.String(nameFlagName, opampBridgeName, "The name of the bridge to use for querying managed collectors.")
	zapFlagSet := flag.NewFlagSet("", flag.ErrorHandling(errorHandling))
	zapCmdLineOpts.BindFlags(zapFlagSet)
	flagSet.AddGoFlagSet(zapFlagSet)
	return flagSet
}

func getHeartbeatInterval(flagSet *pflag.FlagSet) (value time.Duration, changed bool, err error) {
	return getFlagValueAndChanged[time.Duration](flagSet, heartbeatIntervalFlagName)
}

func getConfigFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChanged[string](flagSet, configFilePathFlagName)
}

func getName(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChanged[string](flagSet, nameFlagName)
}

func getKubeConfigFilePath(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChanged[string](flagSet, kubeConfigPathFlagName)
}

func getListenAddr(flagSet *pflag.FlagSet) (value string, changed bool, err error) {
	return getFlagValueAndChanged[string](flagSet, listenAddrFlagName)
}

func getFlagValueAndChanged[T any](flagSet *pflag.FlagSet, flagName string) (value T, changed bool, err error) {
	var zero T
	if changed = flagSet.Changed(flagName); !changed {
		value, err = zero, nil
		return
	}
	switch any(zero).(type) {
	case string:
		val, e := flagSet.GetString(flagName)
		value = any(val).(T)
		err = e
	case time.Duration:
		val, e := flagSet.GetDuration(flagName)
		value = any(val).(T)
		err = e
	default:
		err = fmt.Errorf("unsupported flag type %T", zero)
	}
	return
}
