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

// Flag definitions.
var (
	configFilePathFlag      *string
	listenAddrFlag          *string
	prometheusCREnabledFlag *bool
	kubeConfigPathFlag      *string
	zapCmdLineOpts          zap.Options
)

func initFlags() {
	configFilePathFlag = pflag.String("config-file", DefaultConfigFilePath, "The path to the config file.")
	listenAddrFlag = pflag.String("listen-addr", ":8080", "The address where this service serves.")
	prometheusCREnabledFlag = pflag.Bool("enable-prometheus-cr-watcher", false, "Enable Prometheus CRs as target sources")
	kubeConfigPathFlag = pflag.String("kubeconfig-path", filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file")
	zapCmdLineOpts.BindFlags(flag.CommandLine)
}

func init() {
	initFlags()
}
