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

package logger

import (
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

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Logger.V(4).Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Logger.V(0).Error(nil, fmt.Sprintf(format, v...))
}
