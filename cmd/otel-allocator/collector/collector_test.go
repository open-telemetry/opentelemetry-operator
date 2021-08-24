package collector

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
)

var client Client
var collectors = []string{}

func TestMain(m *testing.M) {
	client = Client{
		k8sClient:     fake.NewSimpleClientset(),
		collectorChan: make(chan []string, 3),
		close:         make(chan struct{}),
	}

	labelMap := map[string]string{
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	}

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	watcher, err := client.k8sClient.CoreV1().Pods("test-ns").Watch(context.Background(), opts)
	if err != nil {
		fmt.Printf("failed to setup a Collector Pod watcher: %v", err)
		os.Exit(1)
	}

	go runWatch(context.Background(), &client, watcher.ResultChan(), map[string]bool{}, func(collectorList []string) { getCollectors(collectorList) })

	code := m.Run()

	close(client.close)

	os.Exit(code)
}

func TestWatchPodAddition(t *testing.T) {
	expected := []string{"test-pod1", "test-pod2", "test-pod3"}

	for _, k := range []string{"test-pod1", "test-pod2", "test-pod3"} {
		expected := pod(k)
		_, err := client.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), expected, metav1.CreateOptions{})
		assert.NoError(t, err)
		collectors = <-client.collectorChan
	}

	assert.Len(t, collectors, 3)

	sort.Strings(collectors)
	assert.Equal(t, collectors, expected)
}

func TestWatchPodDeletion(t *testing.T) {
	expected := []string{"test-pod1"}

	for _, k := range []string{"test-pod2", "test-pod3"} {
		err := client.k8sClient.CoreV1().Pods("test-ns").Delete(context.Background(), k, metav1.DeleteOptions{})
		assert.NoError(t, err)
		collectors = <-client.collectorChan
	}

	assert.Len(t, collectors, 1)

	sort.Strings(collectors)
	assert.Equal(t, collectors, expected)
}

func getCollectors(c []string) {
	collectors = c
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
