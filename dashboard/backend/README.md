# Dashboard — Go API (Phase 1)

Aggregates all four controllers (`01-pod-auto-healer`, `02-rollout-manager`,
`03-resource-optimizer`, `04-autoscaler`) into one REST API: cluster health
from the Kubernetes API, per-controller state from each one's `/metrics`
endpoint, and a live-activity feed synthesized from metric deltas. The
frontend (later phase) never talks to Kubernetes directly — everything
routes through here.

```
Kubernetes API (client-go, read-only)   each controller's /metrics
      │                                       │ (Prometheus text format)
      └──────────────── poller.Run ───────────┘
               (every POLL_INTERVAL, in-memory Cache)
                                │
                    diff vs. previous snapshot
                                │
                    events.Store (ring buffer)
                                │
                          REST handlers
```

## Why this approach

- **No Prometheus server required.** Each controller already exposes
  Prometheus text-format metrics; this API scrapes and parses that
  directly (`internal/promscrape`, `github.com/prometheus/common/expfmt`)
  instead of standing up a Prometheus instance to query.
- **Events from metric deltas, not a second event system.** A counter
  increasing between polls (e.g. `autoscaler_current_replicas` changing)
  is itself the event — no separate instrumentation needed in the
  controllers.
- **In-memory cache and event buffer for Phase 1.** Persistent audit
  history (SQLite/Postgres) is a later phase; handlers read a poller-
  maintained cache so a slow scrape never blocks an HTTP request.

## Run locally

```sh
go run ./cmd/dashboard-api
```

Requires a reachable cluster (`~/.kube/config` or in-cluster) and each
controller's `/metrics` reachable at its configured URL. Config is
env-based:

| Env var | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:8090` | HTTP bind address |
| `POLL_INTERVAL` | `15s` | How often to scrape the cluster and controllers |
| `EVENT_BUFFER_SIZE` | `200` | Max events kept in the in-memory feed |
| `AUTO_HEALER_NAMESPACE` / `_METRICS_URL` | `auto-healer-demo` / cluster-DNS `:8080/metrics` | Project 01 |
| `ROLLOUT_MANAGER_NAMESPACE` / `_METRICS_URL` | `rollout-manager-demo` / ″ | Project 02 |
| `RESOURCE_OPTIMIZER_NAMESPACE` / `_METRICS_URL` | `resource-optimizer-demo` / ″ | Project 03 |
| `AUTOSCALER_NAMESPACE` / `_METRICS_URL` / `AUTOSCALER_DEPLOYMENT` | `autoscaler-demo` / ″ / `demo-app` | Project 04 |

## Endpoints

- `GET /healthz`
- `GET /api/overview` — cluster summary, one-line status per automation, 10 most recent events
- `GET /api/auto-healer` — pods checked, remediations, failures, unhealthy-by-reason
- `GET /api/rollout-manager` — deployments checked, rollbacks, failures, stuck-by-reason
- `GET /api/resource-optimizer` — containers tracked, recommendations, drift entries
- `GET /api/autoscaler` — current/desired replicas, utilization, scale event counts
- `GET /api/events?limit=N&source=X` — recent synthesized events, newest first, optionally filtered to one source (`auto-healer` | `rollout-manager` | `resource-optimizer` | `autoscaler`)

## Deferred to later phases

SSE streaming (`/api/events/stream`), the React/Vite frontend, write
actions (remediate/rollback/apply/scale from the dashboard), and a
persistent audit store — per the project's own phased build order.
