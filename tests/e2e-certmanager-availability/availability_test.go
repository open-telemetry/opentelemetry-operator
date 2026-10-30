// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package certmanageravailability

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const rolloutAnnotation = "opentelemetry.io/cert-manager-availability-restart"

var collectorResource = schema.GroupVersionResource{Group: "opentelemetry.io", Version: "v1beta1", Resource: "opentelemetrycollectors"}

func TestCertManagerAvailabilityRollout(t *testing.T) {
	if os.Getenv("OTEL_CERTMANAGER_E2E") != "1" {
		t.Skip("set OTEL_CERTMANAGER_E2E=1 to run against an isolated test cluster")
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	require.NotEmpty(t, kubeconfig)
	operatorNamespace := os.Getenv("OPERATOR_NAMESPACE")
	if operatorNamespace == "" {
		operatorNamespace = "opentelemetry-operator-system"
	}
	controllerName := os.Getenv("OPERATOR_DEPLOYMENT")
	if controllerName == "" {
		controllerName = "opentelemetry-operator-controller-manager"
	}
	deploymentNames := []string{controllerName}
	if webhookName := os.Getenv("WEBHOOK_DEPLOYMENT"); webhookName != "" && webhookName != controllerName {
		deploymentNames = append(deploymentNames, webhookName)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)
	dynamicClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	deployments := clientset.AppsV1().Deployments(operatorNamespace)
	initial := map[string]string{}
	var serviceAccount string
	for _, name := range deploymentNames {
		deployment, getErr := deployments.Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, getErr)
		require.NotNil(t, deployment.Spec.Replicas)
		require.GreaterOrEqual(t, *deployment.Spec.Replicas, int32(2))
		require.Equal(t, *deployment.Spec.Replicas, deployment.Status.AvailableReplicas)
		if serviceAccount == "" {
			serviceAccount = deployment.Spec.Template.Spec.ServiceAccountName
		}
		require.Equal(t, serviceAccount, deployment.Spec.Template.Spec.ServiceAccountName)
		initial[name] = deployment.Spec.Template.Annotations[rolloutAnnotation]
	}

	namespace, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "certmanager-availability-e2e-"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if cleanupErr := clientset.CoreV1().Namespaces().Delete(context.Background(), namespace.Name, metav1.DeleteOptions{}); cleanupErr != nil && !apierrors.IsNotFound(cleanupErr) {
			t.Errorf("failed to delete namespace %s: %v", namespace.Name, cleanupErr)
		}
	})
	collector := testCollector(namespace.Name)
	_, err = dynamicClient.Resource(collectorResource).Namespace(namespace.Name).Create(ctx, testAdmissionProbe(namespace.Name), metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
	require.ErrorContains(t, err, "cert-manager is not available")

	role, err := clientset.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "certmanager-availability-e2e-"},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"cert-manager.io"},
			Resources: []string{"issuers", "certificaterequests", "certificates"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if cleanupErr := clientset.RbacV1().ClusterRoles().Delete(context.Background(), role.Name, metav1.DeleteOptions{}); cleanupErr != nil && !apierrors.IsNotFound(cleanupErr) {
			t.Errorf("failed to delete ClusterRole %s: %v", role.Name, cleanupErr)
		}
	})
	binding, err := clientset.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "certmanager-availability-e2e-"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: role.Name},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: serviceAccount, Namespace: operatorNamespace}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if cleanupErr := clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), binding.Name, metav1.DeleteOptions{}); cleanupErr != nil && !apierrors.IsNotFound(cleanupErr) {
			t.Errorf("failed to delete ClusterRoleBinding %s: %v", binding.Name, cleanupErr)
		}
	})

	waitForRollouts(t, ctx, clientset, dynamicClient, namespace.Name, operatorNamespace, initial)
	waitForAdmission(t, ctx, dynamicClient, namespace.Name, true)
	_, err = dynamicClient.Resource(collectorResource).Namespace(namespace.Name).Create(ctx, collector, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		if cleanupErr := deleteCollector(cleanupCtx, dynamicClient, namespace.Name); cleanupErr != nil {
			t.Errorf("failed to clean up collector in namespace %s: %v", namespace.Name, cleanupErr)
		}
	})
	certificates := dynamicClient.Resource(schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates"}).Namespace(namespace.Name)
	for _, name := range []string{"availability-ca-cert", "availability-ta-client-cert", "availability-ta-server-cert"} {
		err = wait.PollUntilContextTimeout(ctx, time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			certificate, getErr := certificates.Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(getErr) {
				return false, nil
			}
			if getErr != nil {
				return false, getErr
			}
			conditions, _, nestedErr := unstructured.NestedSlice(certificate.Object, "status", "conditions")
			if nestedErr != nil {
				return false, nestedErr
			}
			for _, condition := range conditions {
				entry, ok := condition.(map[string]any)
				if ok && entry["type"] == "Ready" && entry["status"] == "True" {
					return true, nil
				}
			}
			return false, nil
		})
		require.NoError(t, err, "certificate %s did not become ready", name)
	}
	require.NoError(t, deleteCollector(ctx, dynamicClient, namespace.Name))

	current := map[string]string{}
	for _, name := range deploymentNames {
		deployment, getErr := deployments.Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, getErr)
		current[name] = deployment.Spec.Template.Annotations[rolloutAnnotation]
	}
	require.NoError(t, clientset.RbacV1().ClusterRoleBindings().Delete(ctx, binding.Name, metav1.DeleteOptions{}))
	waitForRollouts(t, ctx, clientset, dynamicClient, namespace.Name, operatorNamespace, current)
	waitForAdmission(t, ctx, dynamicClient, namespace.Name, false)
}

func deleteCollector(ctx context.Context, dynamicClient dynamic.Interface, namespace string) error {
	collectors := dynamicClient.Resource(collectorResource).Namespace(namespace)
	err := collectors.Delete(ctx, "availability", metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return wait.PollUntilContextTimeout(ctx, time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, getErr := collectors.Get(ctx, "availability", metav1.GetOptions{})
		if apierrors.IsNotFound(getErr) {
			return true, nil
		}
		return false, getErr
	})
}

func waitForRollouts(t *testing.T, ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace, operatorNamespace string, initial map[string]string) {
	t.Helper()
	err := wait.PollUntilContextTimeout(ctx, time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		ready := true
		for name, previous := range initial {
			deployment, getErr := clientset.AppsV1().Deployments(operatorNamespace).Get(ctx, name, metav1.GetOptions{})
			if getErr != nil {
				return false, getErr
			}
			if deployment.Status.AvailableReplicas == 0 {
				return false, fmt.Errorf("deployment %s has no available replicas", name)
			}
			if !rolloutComplete(deployment, previous) {
				ready = false
			}
			selector, selectorErr := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
			if selectorErr != nil {
				return false, selectorErr
			}
			pods, listErr := clientset.CoreV1().Pods(operatorNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
			if listErr != nil {
				return false, listErr
			}
			for _, pod := range pods.Items {
				if pod.DeletionTimestamp != nil {
					ready = false
				}
			}
		}
		_, probeErr := dynamicClient.Resource(collectorResource).Namespace(namespace).Create(ctx, testAdmissionProbe(namespace), metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		if probeErr != nil && !strings.Contains(probeErr.Error(), "cert-manager is not available") {
			return false, fmt.Errorf("webhook request failed during rollout: %w", probeErr)
		}
		return ready, nil
	})
	require.NoError(t, err)
}

func waitForAdmission(t *testing.T, ctx context.Context, dynamicClient dynamic.Interface, namespace string, available bool) {
	t.Helper()
	err := wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		_, probeErr := dynamicClient.Resource(collectorResource).Namespace(namespace).Create(ctx, testAdmissionProbe(namespace), metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		if probeErr != nil && !strings.Contains(probeErr.Error(), "cert-manager is not available") {
			return false, probeErr
		}
		return (probeErr == nil) == available, nil
	})
	require.NoError(t, err)
}

func testAdmissionProbe(namespace string) *unstructured.Unstructured {
	collector := testCollector(namespace)
	collector.SetName("availability-probe")
	return collector
}

func rolloutComplete(deployment *appsv1.Deployment, previous string) bool {
	current := deployment.Spec.Template.Annotations[rolloutAnnotation]
	return current != "" && current != previous &&
		deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
		deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
		deployment.Status.Replicas == *deployment.Spec.Replicas
}

func testCollector(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "opentelemetry.io/v1beta1", "kind": "OpenTelemetryCollector",
		"metadata": map[string]any{"name": "availability", "namespace": namespace},
		"spec": map[string]any{
			"mode":            "statefulset",
			"targetAllocator": map[string]any{"enabled": true, "mtls": map[string]any{"enabled": true, "useCertManager": true}},
			"config": map[string]any{
				"receivers": map[string]any{"prometheus": map[string]any{"config": map[string]any{"scrape_configs": []any{map[string]any{"job_name": "test", "static_configs": []any{map[string]any{"targets": []any{"127.0.0.1:8888"}}}}}}}},
				"exporters": map[string]any{"debug": map[string]any{}},
				"service":   map[string]any{"pipelines": map[string]any{"metrics": map[string]any{"receivers": []any{"prometheus"}, "exporters": []any{"debug"}}}},
			},
		},
	}}
}
