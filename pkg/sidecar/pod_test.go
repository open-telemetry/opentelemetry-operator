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

package sidecar_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var _ = Describe("Pod", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should add sidecar when none exists", func() {
		// prepare
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
				},
				// cross-test: the pod has a volume already, make sure we don't remove it
				Volumes: []corev1.Volume{{}},
			},
		}
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "otelcol-sample",
				Namespace: "some-app",
			},
		}
		cfg := config.New(config.WithCollectorImage("some-default-image"))

		// test
		changed, err := sidecar.Add(cfg, logger, otelcol, pod)

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(changed.Spec.Containers).To(HaveLen(2))
		Expect(changed.Spec.Volumes).To(HaveLen(2))
		Expect(changed.Labels["sidecar.opentelemetry.io/injected"]).To(Equal("some-app.otelcol-sample"))
	})

	// this situation should never happen in the current code path, but it should not fail
	// if it's asked to add a new sidecar. The caller is expected to have called ExistsIn before.
	It("should add sidecar, even when one exists", func() {
		// prepare
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
					{Name: naming.Container()},
				},
			},
		}
		otelcol := v1alpha1.OpenTelemetryCollector{}
		cfg := config.New(config.WithCollectorImage("some-default-image"))

		// test
		changed, err := sidecar.Add(cfg, logger, otelcol, pod)

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(changed.Spec.Containers).To(HaveLen(3))
	})

	It("should remove the sidecar", func() {
		// prepare
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
					{Name: naming.Container()},
					{Name: naming.Container()}, // two sidecars! should remove both
				},
			},
		}

		// test
		changed, err := sidecar.Remove(pod)

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(changed.Spec.Containers).To(HaveLen(1))
	})

	It("should not fail to remove when sidecar doesn't exist", func() {
		// prepare
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
				},
			},
		}

		// test
		changed, err := sidecar.Remove(pod)

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(changed.Spec.Containers).To(HaveLen(1))
	})

	DescribeTable("determine whether the pod has a sidecar already", func(expected bool, pod corev1.Pod) {
		Expect(sidecar.ExistsIn(pod)).To(Equal(expected))
	},
		Entry("has-sidecar", true, corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
					{Name: naming.Container()},
				},
			},
		}),
		Entry("does-not-have-sidecar", false, corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		}),
	)
})
