# 05 — Kubernetes Cost Optimizer

Same live usage-vs-configured sampling pipeline as
[03-resource-optimizer](../03-resource-optimizer), but scored in dollars
instead of drift percentage: ranks the most expensive over-provisioned
containers by estimated $/month wasted on requested CPU/memory beyond
what's actually used. Read-only, no patching. Requires
[metrics-server](https://github.com/kubernetes-sigs/metrics-server) running
in the cluster.

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
        cost.MonthlyWaste(configured request, recommended request)
                                │
              Prometheus gauges + ranked top-N log report
```

## Why this approach

- **Cost, not drift percentage.** 03-resource-optimizer flags *any*
  request/limit that has drifted from the recommendation, over- or
  under-provisioned. This project asks a narrower, FinOps-shaped question:
  of the over-provisioned containers, which ones are actually burning the
  most money? A 10x-oversized request on a container with a tiny CPU
  request costs less than a 20%-oversized request on a huge one.
- **Scored on requests, not limits.** A request is what the scheduler
  reserves capacity for and what a cluster autoscaler sizes nodes
  against — that's the number that costs money every idle hour. An
  over-set limit risks nothing but a slower OOM kill.
- **Reuses 03's sampler/analyzer verbatim.** Same percentile-based
  recommender (p50 usage → recommended request), same bounded in-memory
  window — no reason to re-derive "what should this container's request
  be" differently just because this project reports it differently.
- **Illustrative rates, not a specific cloud's price list.** `CPU_CORE_HOUR_RATE`
  / `MEM_GIB_HOUR_RATE` default to blended general-purpose on-demand
  pricing; override them to match actual billing (spot, reserved,
  provider-specific) for a real estimate.

## Run locally

```sh
go run ./cmd/cost-optimizer
```

Requires a reachable cluster with metrics-server installed
(`~/.kube/config` or in-cluster). Config is env-based:

| Env var | Default | Meaning |
|---|---|---|
| `TARGET_NAMESPACE` | `cost-optimizer-demo` | Namespace to analyze |
| `POLL_INTERVAL` | `30s` | How often to sample usage |
| `WINDOW_SIZE` | `60` | Max samples kept per container |
| `MIN_SAMPLES` | `5` | Samples required before recommending |
| `WINDOW_TTL` | `15m` | How long an idle container's window is kept before GC |
| `CPU_REQUEST_MARGIN` | `1.1` | Multiplier applied to p50 CPU for the request recommendation |
| `CPU_LIMIT_MARGIN` | `1.3` | Multiplier applied to p90 CPU for the limit recommendation |
| `MEM_REQUEST_MARGIN` | `1.1` | Multiplier applied to p50 memory for the request recommendation |
| `MEM_LIMIT_MARGIN` | `1.3` | Multiplier applied to p90 memory for the limit recommendation |
| `CPU_CORE_HOUR_RATE` | `0.033` | $ per CPU core-hour used to price waste |
| `MEM_GIB_HOUR_RATE` | `0.0045` | $ per GiB-hour used to price waste |
| `TOP_N` | `5` | Most expensive containers logged per poll |
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

- `cost_optimizer_recommended_{cpu_request,cpu_limit}_millicores{namespace,pod,container}`
- `cost_optimizer_recommended_{memory_request,memory_limit}_bytes{namespace,pod,container}`
- `cost_optimizer_waste_dollars_per_month{namespace,pod,container}`
- `cost_optimizer_total_waste_dollars_per_month`
- `cost_optimizer_containers_tracked`, `cost_optimizer_samples_collected_total`
