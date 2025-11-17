# OpenTelemetry Collector with Target Allocator Setup

This directory contains a production-ready setup for deploying an OpenTelemetry Collector with Target Allocator support on Kubernetes.

## Overview

The Target Allocator is a component that distributes Prometheus scrape targets across a fleet of OpenTelemetry Collector instances. This enables:

- **Horizontal scaling** of metric collection across multiple collector instances
- **Dynamic target discovery** via Prometheus Operator CRDs (ServiceMonitor, PodMonitor, ScrapeConfig)
- **Automatic load balancing** of scrape targets
- **High availability** with consistent hashing or least-weighted strategies

## Architecture

```
┌─────────────────────┐
│  Target Allocator   │◄──── Discovers ServiceMonitors, PodMonitors
│                     │
│  Allocation         │
│  Strategy:          │
│  - consistent-hash  │
│  - least-weighted   │
│  - per-node         │
└──────────┬──────────┘
           │
           │ Assigns targets
           │
    ┌──────┴──────────────────┬─────────────┐
    │                         │             │
    ▼                         ▼             ▼
┌─────────┐            ┌─────────┐    ┌─────────┐
│Collector│            │Collector│    │Collector│
│  Pod 1  │            │  Pod 2  │    │  Pod 3  │
└────┬────┘            └────┬────┘    └────┬────┘
     │                      │              │
     └──────────────────────┴──────────────┘
                            │
                    Scrapes metrics from
                    discovered targets
```

## Prerequisites

1. **Kubernetes Cluster** (v1.21+)
2. **OpenTelemetry Operator** installed
   ```bash
   kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
   ```
3. **Prometheus Operator** (optional, but recommended for ServiceMonitor/PodMonitor support)
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
   ```

## File Structure

```
target-allocator-setup/
├── 00-namespace.yaml                        # Namespace creation
├── 01-rbac.yaml                             # RBAC for collector and target allocator
├── 02-collector.yaml                        # OpenTelemetryCollector CR with target allocator
├── 03-sample-app-with-servicemonitor.yaml   # Example application with ServiceMonitor
└── README.md                                # This file
```

## Quick Start

### 1. Create Namespace
```bash
kubectl apply -f 00-namespace.yaml
```

### 2. Set Up RBAC
```bash
kubectl apply -f 01-rbac.yaml
```

This creates:
- **Service Accounts**: `collector` and `ta`
- **ClusterRoles**: Permissions for target discovery and metric scraping
- **ClusterRoleBindings**: Binds roles to service accounts

### 3. Deploy OpenTelemetry Collector
```bash
kubectl apply -f 02-collector.yaml
```

This deploys:
- **OpenTelemetry Collector StatefulSet** (3 replicas)
- **Target Allocator Deployment**
- **Services** for both components

### 4. (Optional) Deploy Sample Application
```bash
kubectl apply -f 03-sample-app-with-servicemonitor.yaml
```

This creates:
- Sample metrics application (3 replicas)
- Service for the application
- ServiceMonitor for automatic discovery
- PodMonitor example
- ScrapeConfig example

## Verification

### Check Collector Pods
```bash
kubectl get pods -n opentelemetry
```

Expected output:
```
NAME                                           READY   STATUS    RESTARTS   AGE
otel-collector-0                               1/1     Running   0          2m
otel-collector-1                               1/1     Running   0          2m
otel-collector-2                               1/1     Running   0          2m
otel-collector-targetallocator-xxxxxxxxx-xxxxx 1/1     Running   0          2m
```

### Check Target Allocator Logs
```bash
kubectl logs -n opentelemetry -l app.kubernetes.io/component=opentelemetry-targetallocator
```

Look for messages about discovered targets and allocations.

### Access Target Allocator UI
```bash
kubectl port-forward -n opentelemetry svc/otel-collector-targetallocator 8080:80
```

Then open http://localhost:8080 in your browser to see:
- Dashboard with stats
- Target allocation distribution
- Collector status
- Job and target lists

### Verify Target Allocation
```bash
# Get targets from Target Allocator API
kubectl port-forward -n opentelemetry svc/otel-collector-targetallocator 8080:80

# In another terminal:
curl http://localhost:8080/jobs
curl http://localhost:8080/scrape_configs
```

### Check Collector Metrics
```bash
kubectl port-forward -n opentelemetry otel-collector-0 8888:8888

# In another terminal:
curl http://localhost:8888/metrics
```

## Configuration Options

### Allocation Strategies

Choose the allocation strategy based on your use case:

#### 1. Consistent Hashing (Default)
```yaml
targetAllocator:
  allocationStrategy: consistent-hashing
```
- Distributes targets evenly
- Minimal reassignment when collectors scale
- Best for general use cases

#### 2. Least Weighted
```yaml
targetAllocator:
  allocationStrategy: least-weighted
```
- Assigns targets to least loaded collector
- Better load balancing for varied target sizes
- More reassignments during scaling

#### 3. Per-Node
```yaml
targetAllocator:
  allocationStrategy: per-node
mode: daemonset  # Use DaemonSet mode
```
- One collector per Kubernetes node
- Scrapes targets on the same node
- Reduces network traffic
- Best for node-level metrics (kubelet, cAdvisor)

### Prometheus CR Discovery

Filter which ServiceMonitors/PodMonitors to discover:

```yaml
targetAllocator:
  prometheusCR:
    enabled: true
    scrapeInterval: 30s

    # Only discover ServiceMonitors with this label
    serviceMonitorSelector:
      matchLabels:
        monitoring: enabled

    # Only discover PodMonitors in these namespaces
    podMonitorNamespaceSelector:
      matchNames:
        - opentelemetry
        - production
```

### Resource Configuration

Adjust resources based on your cluster size:

```yaml
# For small clusters (< 1000 targets)
targetAllocator:
  resources:
    requests:
      memory: 128Mi
      cpu: 100m
    limits:
      memory: 512Mi
      cpu: 500m

# For large clusters (> 5000 targets)
targetAllocator:
  resources:
    requests:
      memory: 512Mi
      cpu: 500m
    limits:
      memory: 2Gi
      cpu: 2000m
```

## RBAC Explained

### Target Allocator Permissions

The Target Allocator needs to:
- **Discover services** (pods, services, endpoints)
- **Watch Prometheus CRDs** (ServiceMonitor, PodMonitor, ScrapeConfig)
- **Access cluster metadata** (nodes, namespaces)

### Collector Permissions

The Collector needs to:
- **Scrape metrics endpoints** (/metrics)
- **Access Kubernetes metadata** for enrichment
- **Connect to discovered targets**

### Security Best Practices

1. **Use dedicated service accounts** - Separate SA for collector and target allocator
2. **Apply least privilege** - Only grant necessary permissions
3. **Enable security contexts** - Run as non-root user
4. **Use network policies** - Restrict traffic between components
5. **Enable TLS/mTLS** - Secure communication (see mTLS example in E2E tests)

## Troubleshooting

### No Targets Discovered

**Issue**: Target Allocator shows 0 targets

**Solutions**:
1. Check RBAC permissions:
   ```bash
   kubectl auth can-i list servicemonitors --as=system:serviceaccount:opentelemetry:ta
   ```

2. Verify Prometheus Operator CRDs are installed:
   ```bash
   kubectl get crd servicemonitors.monitoring.coreos.com
   ```

3. Check ServiceMonitor labels match selector:
   ```bash
   kubectl get servicemonitor -n opentelemetry -o yaml
   ```

### Targets Not Being Scraped

**Issue**: Targets allocated but no metrics collected

**Solutions**:
1. Check collector can reach target allocator:
   ```bash
   kubectl exec -n opentelemetry otel-collector-0 -- curl http://otel-collector-targetallocator/jobs
   ```

2. Verify collector RBAC has scraping permissions
3. Check target endpoint is accessible:
   ```bash
   kubectl exec -n opentelemetry otel-collector-0 -- curl http://target:port/metrics
   ```

### High Memory Usage

**Issue**: Target Allocator or Collector using too much memory

**Solutions**:
1. Reduce scrape frequency
2. Increase resource limits
3. Scale collector replicas
4. Filter discovered targets with selectors
5. Enable metric relabeling to drop unwanted metrics

### Uneven Target Distribution

**Issue**: Some collectors have many more targets than others

**Solutions**:
1. Switch to `least-weighted` allocation strategy
2. Ensure collectors are all healthy and ready
3. Check for network issues between TA and collectors

## Advanced Examples

### mTLS Between Target Allocator and Collector

See the E2E test example:
```
tests/e2e-ta-collector-mtls/ta-collector-mtls/
```

### Multi-Namespace Discovery

```yaml
targetAllocator:
  prometheusCR:
    enabled: true
    serviceMonitorNamespaceSelector:
      matchExpressions:
        - key: monitoring
          operator: In
          values: [enabled, prometheus]
```

### Custom Scrape Configs

You can still use static scrape configs alongside target allocator:

```yaml
config: 
  receivers:
    prometheus:
      config:
        scrape_configs:
          # Static config (not managed by TA)
          - job_name: 'static-targets'
            static_configs:
              - targets: ['target1:9090', 'target2:9090']

          # Dynamic configs managed by TA
          # Will be automatically populated
```

## Monitoring the Target Allocator

The Target Allocator exposes its own metrics at `:8080/metrics`:

Key metrics:
- `opentelemetry_allocator_targets` - Number of targets per collector
- `opentelemetry_allocator_collectors_discovered` - Number of collector instances
- `opentelemetry_allocator_time_to_allocate` - Time taken to allocate targets

Add a ServiceMonitor for the Target Allocator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: target-allocator
  namespace: opentelemetry
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: opentelemetry-targetallocator
  endpoints:
    - port: targetallocation
      interval: 30s
```

## Performance Tuning

### Large-Scale Deployments

For clusters with many targets (> 10,000):

1. **Increase collector replicas**:
   ```yaml
   replicas: 10
   ```

2. **Tune batch processor**:
   ```yaml
   processors:
     batch:
       timeout: 5s
       send_batch_size: 2048
   ```

3. **Enable compression**:
   ```yaml
   exporters:
     otlp:
       compression: gzip
   ```

4. **Use persistent volumes** for statefulsets to survive pod restarts

### Network Optimization

1. Use `per-node` strategy with DaemonSet for node metrics
2. Enable HTTP/2 and compression
3. Co-locate collectors with targets when possible
4. Use service mesh (Istio, Linkerd) for observability

## References

- [OpenTelemetry Operator Documentation](https://github.com/open-telemetry/opentelemetry-operator)
- [Target Allocator Design Doc](https://github.com/open-telemetry/opentelemetry-operator/blob/main/cmd/otel-allocator/README.md)
- [Prometheus Operator API](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/api.md)
- [E2E Test Examples](../../tests/e2e-targetallocator/)

## Support

For issues and questions:
- GitHub Issues: https://github.com/open-telemetry/opentelemetry-operator/issues
- Slack: #otel-operator on CNCF Slack
