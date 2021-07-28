package collector

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collectors := []string{}

	client := Client{
		k8sClient:     fake.NewSimpleClientset(),
		collectorChan: make(chan []string),
	}

	labelMap := map[string]string{
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	}

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	client.Watch(ctx, labelMap, func(collectorList []string) {})
	// adding sleep to alow the collector watch function to start
	time.Sleep(1 * time.Second)

	t.Run("should create pods", func(t *testing.T) {
		expected := pod("test-pod1")
		_, err := client.k8sClient.CoreV1().Pods("test-ns").Create(ctx, expected, metav1.CreateOptions{})
		assert.NoError(t, err)

		collectors = <-client.collectorChan
		pods, err := client.k8sClient.CoreV1().Pods(ns).List(ctx, opts)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		assert.Equal(t, pods.Items[0].Name, "test-pod1")

	})

	t.Run("should update collector list on pod addition", func(t *testing.T) {
		expected := []string{"test-pod1", "test-pod2", "test-pod3", "test-pod4"}

		for _, k := range []string{"test-pod2", "test-pod3", "test-pod4"} {
			expected := pod(k)
			_, err := client.k8sClient.CoreV1().Pods("test-ns").Create(ctx, expected, metav1.CreateOptions{})
			assert.NoError(t, err)
			collectors = <-client.collectorChan
		}

		assert.Len(t, collectors, 4)

		sort.Strings(collectors)
		assert.Equal(t, collectors, expected)
	})

	t.Run("should update collector list on pod deletion", func(t *testing.T) {
		expected := []string{"test-pod1"}

		for _, k := range []string{"test-pod2", "test-pod3", "test-pod4"} {
			err := client.k8sClient.CoreV1().Pods("test-ns").Delete(ctx, k, metav1.DeleteOptions{})
			assert.NoError(t, err)
			collectors = <-client.collectorChan
		}

		assert.Len(t, collectors, 1)

		sort.Strings(collectors)
		assert.Equal(t, collectors, expected)
	})
}

func pod(name string) *v1.Pod {
	labels := make(map[string]string)
	labels["app.kubernetes.io/instance"] = "default.test"
	labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labels,
		},
	}
}
