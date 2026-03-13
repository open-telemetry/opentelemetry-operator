// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampsrv

import (
	"context"
	"log"

	"github.com/open-telemetry/opamp-go/client/types"
)

var _ types.Logger = &Logger{}

type Logger struct {
	logger *log.Logger
}

func (l *Logger) Debugf(ctx context.Context, format string, v ...any) {
	l.logger.Printf(format, v...)
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...any) {
	l.logger.Printf(format, v...)
}
