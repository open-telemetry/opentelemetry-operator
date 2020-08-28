package controllers_test

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/reconcile"
)

var _ = Describe("OpenTelemetryCollector controller", func() {
	logger := logf.Log.WithName("unit-tests")
	cfg := config.New()
	cfg.FlagSet().Parse([]string{})

	It("should generate the underlying objects on reconciliation", func() {
		// prepare
		nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
		reconciler := controllers.NewReconciler(controllers.Params{
			Client: k8sClient,
			Log:    logger,
			Scheme: scheme.Scheme,
			Config: cfg,
		})
		created := &v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Mode: v1alpha1.ModeDeployment,
			},
		}
		err := k8sClient.Create(context.Background(), created)
		Expect(err).ToNot(HaveOccurred())

		// test
		req := k8sreconcile.Request{
			NamespacedName: nsn,
		}
		_, err = reconciler.Reconcile(req)

		// verify
		Expect(err).ToNot(HaveOccurred())

		// the base query for the underlying objects
		opts := []client.ListOption{
			client.InNamespace(nsn.Namespace),
			client.MatchingLabels(map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			}),
		}

		// verify that we have at least one object for each of the types we create
		// whether we have the right ones is up to the specific tests for each type
		{
			list := &corev1.ConfigMapList{}
			err = k8sClient.List(context.Background(), list, opts...)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).ToNot(BeEmpty())
		}
		{
			list := &corev1.ServiceAccountList{}
			err = k8sClient.List(context.Background(), list, opts...)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).ToNot(BeEmpty())
		}
		{
			list := &corev1.ServiceList{}
			err = k8sClient.List(context.Background(), list, opts...)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).ToNot(BeEmpty())
		}
		{
			list := &appsv1.DeploymentList{}
			err = k8sClient.List(context.Background(), list, opts...)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).ToNot(BeEmpty())
		}
		{
			list := &appsv1.DaemonSetList{}
			err = k8sClient.List(context.Background(), list, opts...)
			Expect(err).ToNot(HaveOccurred())
			// attention! we expect daemonsets to be empty in the default configuration
			Expect(list.Items).To(BeEmpty())
		}

		// cleanup
		Expect(k8sClient.Delete(context.Background(), created)).ToNot(HaveOccurred())
	})

	It("should continue when a task's failure can be recovered", func() {
		// prepare
		taskCalled := false
		reconciler := controllers.NewReconciler(controllers.Params{
			Log: logger,
			Tasks: []controllers.Task{
				{
					Name: "should-fail",
					Do: func(context.Context, reconcile.Params) error {
						return errors.New("should fail!")
					},
					BailOnError: false,
				},
				{
					Name: "should-be-called",
					Do: func(context.Context, reconcile.Params) error {
						taskCalled = true
						return nil
					},
				},
			},
		})

		// test
		err := reconciler.RunTasks(context.Background(), reconcile.Params{})

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(taskCalled).To(BeTrue())
	})

	It("should not continue when a task's failure can't be recovered", func() {
		// prepare
		taskCalled := false
		expectedErr := errors.New("should fail!")
		nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
		reconciler := controllers.NewReconciler(controllers.Params{
			Client: k8sClient,
			Log:    logger,
			Scheme: scheme.Scheme,
			Config: cfg,
			Tasks: []controllers.Task{
				{
					Name: "should-fail",
					Do: func(context.Context, reconcile.Params) error {
						taskCalled = true
						return expectedErr
					},
					BailOnError: true,
				},
				{
					Name: "should-not-be-called",
					Do: func(context.Context, reconcile.Params) error {
						Fail("should not have been called")
						return nil
					},
				},
			},
		})
		created := &v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}
		err := k8sClient.Create(context.Background(), created)
		Expect(err).ToNot(HaveOccurred())

		// test
		req := k8sreconcile.Request{
			NamespacedName: nsn,
		}
		_, err = reconciler.Reconcile(req)

		// verify
		Expect(err).To(MatchError(expectedErr))
		Expect(taskCalled).To(BeTrue())

		// cleanup
		Expect(k8sClient.Delete(context.Background(), created)).ToNot(HaveOccurred())
	})

	It("should skip when the instance doesn't exist", func() {
		// prepare
		nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
		reconciler := controllers.NewReconciler(controllers.Params{
			Client: k8sClient,
			Log:    logger,
			Scheme: scheme.Scheme,
			Config: cfg,
			Tasks: []controllers.Task{
				{
					Name: "should-not-be-called",
					Do: func(context.Context, reconcile.Params) error {
						Fail("should not have been called")
						return nil
					},
				},
			},
		})

		// test
		req := k8sreconcile.Request{
			NamespacedName: nsn,
		}
		_, err := reconciler.Reconcile(req)

		// verify
		Expect(err).ToNot(HaveOccurred())
	})

	It("should be able to register with the manager", func() {
		Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")
		// prepare
		mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
		reconciler := controllers.NewReconciler(controllers.Params{})

		// test
		err = reconciler.SetupWithManager(mgr)

		// verify
		Expect(err).ToNot(HaveOccurred())
	})
})
