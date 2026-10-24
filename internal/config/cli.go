// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/spf13/pflag"
)

var args = os.Args[1:]

func CreateCLIParser(cfg Config) (*pflag.FlagSet, error) {
	f := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	f.ParseErrorsAllowlist.UnknownFlags = true
	if err := registerFlags(f, cfg); err != nil {
		return nil, err
	}
	return f, nil
}

func ApplyCLI(cfg *Config) error {
	f, err := CreateCLIParser(*cfg)
	if err != nil {
		return err
	}
	if err := f.Parse(args); err != nil {
		return err
	}
	return applyFlags(cfg, f)
}
