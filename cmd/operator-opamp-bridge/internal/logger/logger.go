// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/client/types"
)

var _ types.Logger = &Logger{}

type Logger struct {
	Logger logr.Logger
}

func NewLogger(logger logr.Logger) *Logger {
	return &Logger{Logger: logger}
}

func (l *Logger) Debugf(ctx context.Context, format string, v ...interface{}) {
	l.Logger.V(4).Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...interface{}) {
	l.Logger.V(0).Error(nil, fmt.Sprintf(format, v...))
}
