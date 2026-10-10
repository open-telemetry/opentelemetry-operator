// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/spf13/pflag"
)

var args = os.Args[1:]

func CreateCLIParser(cfg Config) *pflag.FlagSet {
	f := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	f.ParseErrorsAllowlist.UnknownFlags = true
	registerFlags(f, cfg)
	return f
}

func ApplyCLI(cfg *Config) error {
	f := CreateCLIParser(*cfg)
	if err := f.Parse(args); err != nil {
		return err
	}

	applyFlags(cfg, f)

	return nil
}
