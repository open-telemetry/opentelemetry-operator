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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstrumentationWebhook(t *testing.T) {
	inst := &Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{AnnotationDefaultAutoInstrumentationJava: "img:1"},
		},
	}
	inst.Default()
	assert.Equal(t, "img:1", inst.Spec.Java.Image)
}
