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

// Package config contains the operator's runtime configuration.
package config

import (
	"sync"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// changeHandler is implemented by any structure that is able to register callbacks
// and call them using one single method.
type changeHandler interface {
	// Do will call every registered callback.
	Do() error
	// Register this function as a callback that will be executed when Do() is called.
	Register(f func() error)
}

// newOnChange returns a thread-safe ChangeHandler.
func newOnChange() changeHandler {
	return &onChange{
		logger: logf.Log.WithName("change-handler"),
	}
}

type onChange struct {
	logger logr.Logger

	callbacks   []func() error
	muCallbacks sync.Mutex
}

func (o *onChange) Do() error {
	o.muCallbacks.Lock()
	defer o.muCallbacks.Unlock()
	for _, fn := range o.callbacks {
		if err := fn(); err != nil {
			o.logger.Error(err, "change callback failed")
		}
	}
	return nil
}

func (o *onChange) Register(f func() error) {
	o.muCallbacks.Lock()
	defer o.muCallbacks.Unlock()
	o.callbacks = append(o.callbacks, f)
}
