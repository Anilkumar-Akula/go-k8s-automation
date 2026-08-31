# 03 — Kubernetes Resource Optimizer

Polls metrics-server for live CPU/memory usage, compares it against each
container's configured requests/limits, and recommends what they should
be — read-only, no patching. Requires [metrics-server](https://github.com/kubernetes-sigs/metrics-server)
running in the cluster (ships by default on most managed clusters; `minikube
addons enable metrics-server` / a Helm install on kind).

```
Kubernetes API (Pod specs)      metrics.k8s.io (PodMetrics, from metrics-server)
      │                                    │
      └──────────────── sampler.Collect ───┘
                         (join by pod/container name)
                                │
                    per-container rolling window
                       (last N usage samples)
                                │
              percentile recommender (p50 → request, p90 → limit)
                                │
              drift check vs. configured request/limit
                                │
              Prometheus gauges + structured log warnings
```

## Why this approach

- **Percentile-based, not a fixed rule of thumb.** Same idea as the
  [Vertical Pod Autoscaler recommender](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler):
  request tracks typical usage (p50), limit covers spikes (p90) — not a
  static "usage + 20%" that ignores actual variance.
- **Recommends, doesn't patch.** Changing live requests/limits can trigger
  Pod restarts (limits are immutable on running Pods pre-in-place-resize);
  this project surfaces the recommendation and lets a human (or a separate,
  explicitly-authorized controller) decide when to apply it.
- **No time-series database.** A bounded in-memory ring buffer per
  container is enough for "usage over the last `WINDOW_SIZE` polls" —
  metrics-server itself only keeps the current reading, so nothing here
  is throwing away durability it could have.

## Run locally

```sh
go run ./cmd/resource-optimizer
```

Requires a reachable cluster with metrics-server installed
(`~/.kube/config` or in-cluster). Config is env-based:

| Env var | Default | Meaning |
|---|---|---|
| `TARGET_NAMESPACE` | `resource-optimizer-demo` | Namespace to analyze |
| `POLL_INTERVAL` | `30s` | How often to sample usage |
| `WINDOW_SIZE` | `60` | Max samples kept per container |
| `MIN_SAMPLES` | `5` | Samples required before recommending |
| `WINDOW_TTL` | `15m` | How long an idle container's window is kept before GC |
| `CPU_REQUEST_MARGIN` | `1.1` | Multiplier applied to p50 CPU for the request recommendation |
| `CPU_LIMIT_MARGIN` | `1.3` | Multiplier applied to p90 CPU for the limit recommendation |
| `MEM_REQUEST_MARGIN` | `1.1` | Multiplier applied to p50 memory for the request recommendation |
| `MEM_LIMIT_MARGIN` | `1.3` | Multiplier applied to p90 memory for the limit recommendation |
| `DRIFT_THRESHOLD` | `0.3` | Fractional difference from current before a resource is flagged |
| `METRICS_ADDR` | `:8080` | Prometheus `/metrics` bind address |

## Deploy

```sh
kubectl apply -f deploy/serviceaccount.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
```

RBAC is a namespaced `Role`, not a `ClusterRole`, and grants only
`get`/`list` on Pods and `metrics.k8s.io` PodMetrics — no write verbs
anywhere, since this service never modifies cluster state.

## Metrics

- `resource_optimizer_recommended_{cpu_request,cpu_limit}_millicores{namespace,pod,container}`
- `resource_optimizer_recommended_{memory_request,memory_limit}_bytes{namespace,pod,container}`
- `resource_optimizer_drift_detected_total{namespace,pod,container,resource,field,direction}`
- `resource_optimizer_containers_tracked`, `resource_optimizer_samples_collected_total`
