# 04 — Auto-Scaling Controller

Custom horizontal autoscaler for one Deployment: polls metrics-server for
live CPU usage, compares it against configured CPU requests, and scales
`spec.replicas` the same way the built-in HorizontalPodAutoscaler does —
`desired = ceil(current * observedUtilization / targetUtilization)`.

```
Kubernetes API (Deployment + Pods)   metrics.k8s.io (PodMetrics)
      │                                       │
      └──────────────── usage.Collect ────────┘
               (avg CPU usage / CPU requests, by label selector)
                                │
                    scaler.Decide (ratio formula, tolerance band)
                                │
                    per-direction cooldown (scale up fast, down slow)
                                │
                    Deployments().UpdateScale (scale subresource only)
```

## Why this approach

- **Same formula as HPA, not a fixed step.** `ceil(current * observed/target)`
  converges in one step instead of nudging replicas up/down by a fixed
  amount and re-polling repeatedly.
- **Scale subresource, not the full Deployment.** `GetScale`/`UpdateScale`
  touch only `spec.replicas` — RBAC grants `deployments/scale` write, not
  `deployments` write, so this can never touch the Pod template.
- **Asymmetric cooldowns.** Scaling up is allowed quickly (absorb load);
  scaling down waits out a longer cooldown (`SCALE_DOWN_COOLDOWN`,
  default 5m) so a momentary dip doesn't thrash replicas — same
  stabilization-window idea as HPA's default behavior.
- **Tolerance band.** A ratio within `TOLERANCE` (default 10%) of 1 is
  treated as "close enough," so it doesn't reconcile on noise every poll.

## Run locally

```sh
go run ./cmd/autoscaler
```

Requires a reachable cluster with metrics-server installed
(`~/.kube/config` or in-cluster). Config is env-based:

| Env var | Default | Meaning |
|---|---|---|
| `TARGET_NAMESPACE` | `autoscaler-demo` | Namespace of the target Deployment |
| `TARGET_DEPLOYMENT` | `demo-app` | Deployment to scale |
| `POLL_INTERVAL` | `15s` | How often to sample usage and reconcile |
| `TARGET_CPU_PERCENT` | `50` | Target average CPU usage, as % of configured CPU request |
| `TOLERANCE` | `0.1` | Fractional band around the target ratio treated as no-op |
| `MIN_REPLICAS` | `1` | Lower bound on replicas |
| `MAX_REPLICAS` | `10` | Upper bound on replicas |
| `SCALE_UP_COOLDOWN` | `60s` | Minimum time between scale-up events |
| `SCALE_DOWN_COOLDOWN` | `300s` | Minimum time between scale-down events |
| `METRICS_ADDR` | `:8080` | Prometheus `/metrics` bind address |

## Deploy

```sh
kubectl apply -f deploy/serviceaccount.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
```

## Metrics

- `autoscaler_current_replicas{namespace,deployment}`
- `autoscaler_desired_replicas{namespace,deployment}`
- `autoscaler_current_utilization_percent{namespace,deployment}`
- `autoscaler_scale_events_total{namespace,deployment,direction}`
