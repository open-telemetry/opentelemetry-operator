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

	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var _ = Describe("Annotation", func() {

	DescribeTable("determine the right effective annotation value",
		func(expected string, pod corev1.Pod, ns corev1.Namespace) {
			// test
			annValue := sidecar.AnnotationValue(ns, pod)

			// verify
			Expect(annValue).To(Equal(expected))
		},

		Entry("pod-true-overrides-ns",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "false",
					},
				},
			},
		),

		Entry("ns-has-concrete-instance",
			"some-instance",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "some-instance",
					},
				},
			},
		),

		Entry("pod-has-concrete-instance",
			"some-instance-from-pod",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "some-instance-from-pod",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "some-instance",
					},
				},
			},
		),

		Entry("pod-has-explicit-false",
			"false",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "false",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "some-instance",
					},
				},
			},
		),

		Entry("pod-has-no-annotations",
			"some-instance",
			corev1.Pod{},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "some-instance",
					},
				},
			},
		),

		Entry("ns-has-no-annotations",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "true",
					},
				},
			},
			corev1.Namespace{},
		),
	)

})
