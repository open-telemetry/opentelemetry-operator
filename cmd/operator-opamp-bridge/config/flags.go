// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
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
	kubeConfigPathFlagName    = "kubeconfig-path"
	heartbeatIntervalFlagName = "heartbeat-interval"
	nameFlagName              = "name"
	defaultHeartbeatInterval  = 30 * time.Second
)

// We can't bind this flag to our FlagSet, so we need to handle it separately.
var zapCmdLineOpts zap.Options

func GetFlagSet(errorHandling pflag.ErrorHandling) *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(opampBridgeName, errorHandling)
	flagSet.String(configFilePathFlagName, defaultConfigFilePath, "The path to the config file.")
	flagSet.String(listenAddrFlagName, ":8080", "The address where this service serves.")
	flagSet.String(kubeConfigPathFlagName, filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file.")
	flagSet.Duration(heartbeatIntervalFlagName, defaultHeartbeatInterval, "The interval to use for sending a heartbeat. Setting it to 0 disables the heartbeat.")
	flagSet.String(nameFlagName, opampBridgeName, "The name of the bridge to use for querying managed collectors.")
	zapFlagSet := flag.NewFlagSet("", flag.ErrorHandling(errorHandling))
	zapCmdLineOpts.BindFlags(zapFlagSet)
	flagSet.AddGoFlagSet(zapFlagSet)
	return flagSet
}

func getHeartbeatInterval(flagset *pflag.FlagSet) (time.Duration, error) {
	return flagset.GetDuration(heartbeatIntervalFlagName)
}

func getConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(configFilePathFlagName)
}

func getName(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(nameFlagName)
}

func getKubeConfigFilePath(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(kubeConfigPathFlagName)
}

func getListenAddr(flagSet *pflag.FlagSet) (string, error) {
	return flagSet.GetString(listenAddrFlagName)
}
