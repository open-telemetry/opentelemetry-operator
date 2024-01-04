package healthchecker

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/sourcegraph/conc/stream"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
)

type HealthChecker interface {
	SetCollectors(selectors []string)
	GetComponentHealth(selector string) map[string]*protobufs.ComponentHealth
	Close() error
	Start()
}

var _ HealthChecker = &Client{}

type healthcheckTuple struct {
	selector string
	pod      v1.Pod
}

type Client struct {
	log logr.Logger

	// mu protects access to collectors.
	mu sync.RWMutex

	// collectors contains the map from selector to the pods component health.
	collectors map[string]map[string]*protobufs.ComponentHealth

	// healthcheckCh is the channel that holds collector ips to healthcheck.
	healthcheckCh chan healthcheckTuple

	// healthcheckStream reads messages from the channel and processes them.
	healthcheckStream *stream.Stream

	// close closes the channel.
	close chan struct{}

	// httpClient holds the http.Client.
	httpClient http.Client

	// k8sClient allows the checker to query the Kube API for collector pods.
	k8sClient client.Client

	// ticker adds messages to the healthcheckCh.
	ticker *time.Ticker
	cfg    config.HealthCheckConfig
	clock  clock.Clock
}

func NewClient(log logr.Logger, cfg config.HealthCheckConfig, c client.Client) *Client {
	interval := cfg.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &Client{
		log:               log,
		cfg:               cfg,
		clock:             clock.RealClock{},
		healthcheckCh:     make(chan healthcheckTuple),
		healthcheckStream: stream.New().WithMaxGoroutines(10),
		close:             make(chan struct{}),
		k8sClient:         c,
		collectors:        make(map[string]map[string]*protobufs.ComponentHealth),
		httpClient: http.Client{
			Timeout: 10 * time.Second,
		},
		ticker: time.NewTicker(interval),
	}
}

func (c *Client) Start() {
	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.log.V(4).Info("running health checking")
				c.mu.RLock()
				ctx := context.Background()
				for selector := range c.collectors {
					pods, err := c.getPodsForSelector(ctx, selector)
					if err != nil {
						c.log.Error(err, "failed to get pod ips")
						continue
					}
					for _, pod := range pods {
						c.healthcheckCh <- healthcheckTuple{
							selector: selector,
							pod:      pod,
						}
					}
				}
				c.mu.RUnlock()
			case <-c.close:
				c.ticker.Stop()
				c.log.Info("stopping health checking")
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case tuple := <-c.healthcheckCh:
				c.healthcheckStream.Go(func() stream.Callback {
					result, err := c.runHealthcheck(tuple.pod.Status.PodIP, tuple.pod.Status.StartTime.UnixNano())
					if err != nil {
						c.log.Error(err, "error running healthcheck")
						return nil
					}
					c.mu.Lock()
					defer c.mu.Unlock()
					c.collectors[tuple.selector][tuple.pod.GetName()] = result
					return nil
				})
			case <-c.close:
				c.log.Info("stopping health check pool")
				c.healthcheckStream.Wait()
				c.log.Info("finished wait")
			}
		}
	}()
}

func (c *Client) runHealthcheck(ip string, startTimeNanos int64) (*protobufs.ComponentHealth, error) {
	response, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d%s", ip, c.cfg.Port, c.cfg.Path))
	if err != nil {
		return nil, err
	}
	return &protobufs.ComponentHealth{
		Healthy:            response.StatusCode == 200,
		StartTimeUnixNano:  uint64(startTimeNanos),
		LastError:          "",
		Status:             strconv.Itoa(response.StatusCode),
		StatusTimeUnixNano: uint64(c.clock.Now().UnixNano()),
	}, nil
}

func (c *Client) getPodsForSelector(ctx context.Context, selector string) ([]v1.Pod, error) {
	podList := v1.PodList{}
	selMap := map[string]string{}
	for _, kvPair := range strings.Split(selector, ",") {
		kv := strings.Split(kvPair, "=")
		// skip malformed pairs
		if len(kv) != 2 {
			continue
		}
		selMap[kv[0]] = kv[1]
	}
	err := c.k8sClient.List(ctx, &podList, client.MatchingLabels(selMap))
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (c *Client) SetCollectors(selectors []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Set up removal map
	removalCandidates := map[string]struct{}{}
	for collector := range c.collectors {
		removalCandidates[collector] = struct{}{}
	}
	// Check for additions
	for _, selector := range selectors {
		if _, ok := c.collectors[selector]; !ok {
			c.collectors[selector] = make(map[string]*protobufs.ComponentHealth)
		} else {
			delete(removalCandidates, selector)
		}
	}
	for collector := range removalCandidates {
		delete(c.collectors, collector)
	}
}

func (c *Client) GetComponentHealth(selector string) map[string]*protobufs.ComponentHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.collectors[selector]
}

func (c *Client) Close() error {
	c.close <- struct{}{}
	return nil
}
