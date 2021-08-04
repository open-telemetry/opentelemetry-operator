package collector

import (
	"context"

	"log"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	watcherTimeout = 15 * time.Minute
)

var (
	ns = os.Getenv("OTELCOL_NAMESPACE")
)

type Client struct {
	k8sClient     kubernetes.Interface
	collectorChan chan []string
	close         chan struct{}
}

func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return &Client{}, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return &Client{}, err
	}

	return &Client{
		k8sClient: clientset,
		close:     make(chan struct{}),
	}, nil
}

func (k *Client) Watch(ctx context.Context, labelMap map[string]string, fn func(collectors []string)) {
	collectorMap := map[string]bool{}

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	pods, err := k.k8sClient.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	for i := range pods.Items {
		pod := pods.Items[i]
		if pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			collectorMap[pod.Name] = true
		}
	}

	collectorKeys := make([]string, len(collectorMap))
	i := 0
	for k := range collectorMap {
		collectorKeys[i] = k
		i++
	}
	fn(collectorKeys)

	go func() {
		for {
			watcher, err := k.k8sClient.CoreV1().Pods(ns).Watch(ctx, opts)
			if err != nil {
				log.Fatal(err)
			}
			c := watcher.ResultChan()
		Inner:
			for {
				select {
				case <-k.close:
					return
				case <-ctx.Done():
					return
				case event, ok := <-c:
					if !ok {
						log.Fatal(err)
					}

					pod, ok := event.Object.(*v1.Pod)
					if !ok {
						log.Fatal(err)
					}

					switch event.Type {
					case watch.Added:
						collectorMap[pod.Name] = true
					case watch.Deleted:
						delete(collectorMap, pod.Name)
					}

					collectorKeys := make([]string, len(collectorMap))
					i := 0
					for k := range collectorMap {
						collectorKeys[i] = k
						i++
					}
					select {
					case k.collectorChan <- collectorKeys:
					default:
						fn(collectorKeys)
					}
				case <-time.After(watcherTimeout):
					break Inner
				}
			}
		}
	}()
}

func (k *Client) Close() {
	close(k.close)
}
