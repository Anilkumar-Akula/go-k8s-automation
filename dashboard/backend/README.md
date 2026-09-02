# Dashboard — Go API

Aggregates all four controllers (`01-pod-auto-healer`, `02-rollout-manager`,
`03-resource-optimizer`, `04-autoscaler`) into one REST API: cluster health
from the Kubernetes API, per-controller state from each one's `/metrics`
endpoint, a live-activity feed synthesized from metric deltas (Phase 1-4),
and — as of Phase 5 — a small set of typed control actions with validation,
audit logging, and idempotency. The frontend never talks to Kubernetes
directly; everything routes through here.

```
Kubernetes API (client-go)              each controller's /metrics
      │  read (observability)                  │ (Prometheus text format)
      │  write (actions, narrow & typed)        │
      │                                         │
      │              ┌──────── poller.Run ──────┘
      │              │  (every POLL_INTERVAL, in-memory Cache)
      │              │         diff vs. previous → events.Store
      │              ▼
      │         REST read handlers
      │
      └──── POST /api/v1/actions/* ──── validate → execute → audit (SQLite) → metrics
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
- **In-memory cache and event buffer for observability reads.** Handlers
  read a poller-maintained cache so a slow scrape never blocks an HTTP
  request.
- **Actions are a fixed, typed list — never arbitrary Kubernetes writes.**
  `internal/actions` exposes exactly four operations (restart a Pod,
  rollback a Deployment, apply a cached recommendation, set a replica
  count), each with its own request struct and validation. There is no
  path from the API to `kubectl exec`, `apply -f`, or a patch against an
  arbitrary resource.
- **Idempotency by design.** Every action handler accepts an
  `Idempotency-Key` header; a repeated key within the TTL replays the
  cached response instead of re-executing — a double-click or retried
  request can't roll back twice.
- **Audit trail survives restarts.** Every action attempt (success or
  failure) is recorded to SQLite (`internal/audit`, pure-Go driver, no
  cgo) — unlike the in-memory observability event feed, which is fine to
  lose on restart.

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
| `ACTIONS_MAX_REPLICAS` | `20` | Upper bound the scale action's validation enforces |
| `IDEMPOTENCY_TTL` | `10m` | How long an `Idempotency-Key` is remembered |
| `AUDIT_DB_PATH` | `dashboard-audit.db` | SQLite file for the audit trail |
| `ACTOR` | `operator` | Recorded on every audit event — hardcoded until Phase 6 (auth) gives real per-request identity |

## Endpoints

Read (observability):
- `GET /healthz`, `GET /metrics` (this service's own Prometheus metrics)
- `GET /api/v1/overview` — cluster summary, one-line status per automation, 10 most recent events
- `GET /api/v1/auto-healer` — pods checked, remediations, failures, unhealthy-by-reason
- `GET /api/v1/rollout-manager` — deployments checked, rollbacks, failures, stuck-by-reason
- `GET /api/v1/resource-optimizer` — containers tracked, recommendations, drift entries
- `GET /api/v1/autoscaler` — current/desired replicas, utilization, scale event counts
- `GET /api/v1/events?limit=N&source=X` — recent synthesized events, newest first, optionally filtered to one source
- `GET /api/v1/audit?limit=N` — persisted control-action history

Write (control actions — see `internal/actions`):
- `POST /api/v1/actions/healer/restart` `{namespace, pod, reason}`
- `POST /api/v1/actions/rollout/rollback` `{namespace, deployment, reason}`
- `POST /api/v1/actions/optimizer/apply` `{namespace, pod, container, reason}` — applies the currently-cached recommendation for that container
- `POST /api/v1/actions/autoscaler/scale` `{namespace, deployment, replicas, reason}`

All four validate their input, execute through `internal/actions.Executor`,
write an `internal/audit.Event`, and update the `dashboard_actions_*`
Prometheus metrics — success or failure. Pass an `Idempotency-Key` header
to make a retry safe.

## Deferred to later phases

SSE streaming, the confirmation UI is client-side only (no server-side
"pending approval" step), and authentication/RBAC (`internal/actions` is
reachable by anyone who can reach this API right now — Phase 6 adds JWT
and per-role enforcement) — per the project's own phased build order.
