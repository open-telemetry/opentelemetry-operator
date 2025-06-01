// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var args = os.Args[1:]

func ApplyCLI(cfg *Config) error {

	f := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	f.ParseErrorsWhitelist.UnknownFlags = true
	f.Bool(constants.FlagMulti, false, "")
	f.Bool(constants.FlagApacheHttpd, false, "")
	f.Bool(constants.FlagDotNet, false, "")
	f.Bool(constants.FlagGo, false, "")
	f.Bool(constants.FlagPython, false, "")
	f.Bool(constants.FlagNginx, false, "")
	f.Bool(constants.FlagNodeJS, false, "")
	f.Bool(constants.FlagJava, false, "")
	f.String("ignore-missing-collector-crds", "", "")
	f.String("collector-image", "", "")
	f.String("target-allocator-image", "", "")
	f.String("operator-opamp-bridge-image", "", "")
	f.String("auto-instrumentation-java-image", "", "")
	f.String("auto-instrumentation-nodejs-image", "", "")
	f.String("auto-instrumentation-python-image", "", "")
	f.String("auto-instrumentation-dotnet-image", "", "")
	f.String("auto-instrumentation-go-image", "", "")
	f.String("auto-instrumentation-apache-httpd-image", "", "")
	f.String("labels-filter", "", "")
	f.String("annotations-filter", "", "")
	err := f.Parse(args)
	if err != nil {
		return err
	}

	f.Visit(func(fl *pflag.Flag) {
		if fl.Changed {
			switch fl.Name {
			case constants.FlagMulti:
				cfg.EnableMultiInstrumentation, _ = f.GetBool("enable-multi-instrumentation")
			case constants.FlagApacheHttpd:
				cfg.EnableApacheHttpdInstrumentation, _ = f.GetBool(constants.FlagApacheHttpd)
			case constants.FlagDotNet:
				cfg.EnableDotNetAutoInstrumentation, _ = f.GetBool(constants.FlagDotNet)
			case constants.FlagGo:
				cfg.EnableGoAutoInstrumentation, _ = f.GetBool(constants.FlagGo)
			case constants.FlagPython:
				cfg.EnablePythonAutoInstrumentation, _ = f.GetBool(constants.FlagPython)
			case constants.FlagNginx:
				cfg.EnableNginxAutoInstrumentation, _ = f.GetBool(constants.FlagNginx)
			case constants.FlagNodeJS:
				cfg.EnableNodeJSAutoInstrumentation, _ = f.GetBool(constants.FlagNodeJS)
			case constants.FlagJava:
				cfg.EnableJavaAutoInstrumentation, _ = f.GetBool(constants.FlagJava)
			case "ignore-missing-collector-crds":
				cfg.IgnoreMissingCollectorCRDs, _ = f.GetBool("ignore-missing-collector-crds")
			case "collector-image":
				cfg.CollectorImage, _ = f.GetString("collector-image")
			case "target-allocator-image":
				cfg.TargetAllocatorImage, _ = f.GetString("target-allocator-image")
			case "operator-opamp-bridge-image":
				cfg.OperatorOpAMPBridgeImage, _ = f.GetString("operator-opamp-bridge-image")
			case "auto-instrumentation-java-image":
				cfg.AutoInstrumentationJavaImage, _ = f.GetString("auto-instrumentation-java-image")
			case "auto-instrumentation-nodejs-image":
				cfg.AutoInstrumentationNodeJSImage, _ = f.GetString("auto-instrumentation-nodejs-image")
			case "auto-instrumentation-python-image":
				cfg.AutoInstrumentationPythonImage, _ = f.GetString("auto-instrumentation-python-image")
			case "auto-instrumentation-dotnet-image":
				cfg.AutoInstrumentationDotNetImage, _ = f.GetString("auto-instrumentation-dotnet-image")
			case "auto-instrumentation-go-image":
				cfg.AutoInstrumentationGoImage, _ = f.GetString("auto-instrumentation-go-image")
			case "auto-instrumentation-apache-httpd-image":
				cfg.AutoInstrumentationApacheHttpdImage, _ = f.GetString("auto-instrumentation-apache-httpd-image")
			case "auto-instrumentation-nginx-image":
				cfg.AutoInstrumentationNginxImage, _ = f.GetString("auto-instrumentation-nginx-image")
			case "labels-filter":
				cfg.LabelsFilter, _ = f.GetStringSlice("labels-filter")
			case "annotations-filter":
				cfg.AnnotationsFilter, _ = f.GetStringSlice("annotations-filter")
			}
		}
	})

	return nil
}
