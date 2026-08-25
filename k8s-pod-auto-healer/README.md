# k8s-pod-auto-healer

**Status: COMPLETE ✅**

Watches Pods in a target namespace, detects CrashLoopBackOff / ImagePullBackOff /
OOMKilled / Failed, and remediates by deleting the pod (its controller replaces
it) — capped at `MAX_REMEDIATION_ATTEMPTS` per owning workload with exponential
backoff. Runs as a Deployment inside the cluster it watches, with least-privilege
RBAC, Prometheus metrics, and graceful shutdown.

## Architecture

```
                    Kubernetes API
                          │
                    Watch (Pods, target ns)
                          │
              ┌───────────┴───────────┐
              │   self-reconnecting    │  internal/watcher
              │   watch loop           │
              └───────────┬───────────┘
                          │ event (pod, type)
                          ▼
              ┌───────────────────────┐
              │   health detector      │  internal/detector
              │   Failed / CrashLoop /  │
              │   ImagePullBackOff /    │
              │   OOMKilled             │
              └───────────┬───────────┘
                          │ unhealthy
                          ▼
              ┌───────────────────────┐
              │  in-flight dedup guard │  cmd/auto-healer
              │  (1 goroutine/pod)     │
              └───────────┬───────────┘
                          ▼
              ┌───────────────────────┐
              │  owner resolution      │  internal/remediation/owner.go
              │  Pod → ReplicaSet →    │
              │  Deployment | StatefulSet │
              │  refuse: bare/Job/DaemonSet │
              └───────────┬───────────┘
                          ▼
              ┌───────────────────────┐
              │  retrier                │  internal/remediation/retry.go
              │  per-workload counter,  │
              │  exp. backoff, TTL GC   │
              └───────────┬───────────┘
                          ▼
              ┌───────────────────────┐
              │  delete pod             │  internal/remediation/remediator.go
              └───────────────────────┘

Prometheus metrics + graceful SIGTERM shutdown wrap the whole loop (cmd/auto-healer/main.go).
```

## Layout

```
cmd/auto-healer/       entrypoint, wiring, graceful shutdown
internal/config/       env-based config
internal/kclient/      in-cluster / kubeconfig client
internal/watcher/      self-reconnecting pod watch
internal/detector/     health checks
internal/remediation/  owner resolution, retry+backoff, delete
internal/metrics/      Prometheus metrics
deploy/                ServiceAccount, Role/RoleBinding, Deployment
```

## Failure detection

`internal/detector.CheckPodHealth` flags a pod unhealthy on the first match:

- `pod.Status.Phase == Failed`
- container waiting reason contains `CrashLoopBackOff`
- container waiting reason contains `ImagePullBackOff`
- container's last termination reason is `OOMKilled`

Restart count (`RESTART_THRESHOLD`, default 3) is logged only — it's not itself
a remediation trigger, since CrashLoopBackOff already covers "keeps restarting."
`DELETED` watch events and pods that already have `DeletionTimestamp` set are
skipped outright (nothing to remediate on something already gone or dying).

## Remediation flow

1. Watch event → unhealthy → resolve owning workload
   (`internal/remediation.ResolveWorkloadKey`): walks `Pod.OwnerReferences`,
   and for a `ReplicaSet` owner, fetches the RS to find its `Deployment` owner.
   Refuses (no delete) on: no owner (bare pod), `Job`, `DaemonSet` — a Job
   restart changes completion semantics, a bare pod delete is permanent.
2. Retry key is the **workload** (`Deployment/ns/name`, `StatefulSet/ns/name`),
   not the pod name — a Deployment gives every replacement pod a new name, so
   per-pod-name tracking would silently reset the counter on every crash.
3. `Retrier.Attempt(key)` returns the attempt number + backoff, or rejects
   once `MAX_REMEDIATION_ATTEMPTS` is exceeded for that workload.
4. Sleep the backoff, then delete the pod (`internal/remediation.Remediate`).

## Retry / backoff

`Backoff(initial, attempt) = initial * 2^(attempt-1)` — default 2s → 4s → 8s.
Verified live against a real crash loop: attempt 1/2/3 fired at the correct
spacing, then `max remediation attempts reached` and no further deletes —
even across pod replacement (new pod name, same workload key, counter held).

Retry state is an in-memory map keyed by workload, garbage-collected on
`RETRY_TTL` (default 10m) so a long-running process doesn't leak memory over
the cluster's lifetime.

## Concurrency protection

Two real races surfaced during live testing against an actual crash-looping
Deployment (not caught by unit tests, which don't exercise real watch timing):

- **Duplicate remediation goroutines** — a single unhealthy pod fires several
  `MODIFIED` events in quick succession (each container-status field change
  is its own event). The handler spawned one goroutine per event with no
  dedup, so 3 goroutines raced for the same pod and burned the entire retry
  budget in ~3 seconds instead of pacing real attempts. Fixed with an
  in-flight guard (`sync.Map` keyed by `namespace/name`) — only one
  remediation goroutine per pod at a time.
- **Termination-window race** — `Delete()` only marks a pod for termination
  (default 30s grace period); the container can keep restarting inside that
  window and fire more `CrashLoopBackOff` events for a pod already on its way
  out, spawning a wasted extra attempt. Fixed by skipping any pod with
  `DeletionTimestamp != nil`.

Final verification after both fixes: clean one-attempt-per-pod cycle, zero
duplicate goroutines, zero `not found` 404s from racing a delete against an
already-terminating pod.

## Metrics

Prometheus, served on `METRICS_ADDR` (default `:8080`) at `/metrics`:

- `auto_healer_pods_checked_total`
- `auto_healer_unhealthy_pods_total{reason}`
- `auto_healer_remediation_total`
- `auto_healer_remediation_failures_total`
- `auto_healer_watch_reconnects_total`
- `auto_healer_remediation_duration_seconds` (histogram)

## Docker

Multi-stage build, `gcr.io/distroless/static-debian12:nonroot` runtime image.
Runs as numeric UID `65532:65532` — a named `USER nonroot:nonroot` fails
Kubernetes' `runAsNonRoot` check because distroless has no `/etc/passwd` to
resolve the name against; the fix is the explicit numeric UID.

```
docker build -t pod-auto-healer:latest .
```

## RBAC

Namespaced `Role` + `RoleBinding`, not `ClusterRole` — the healer only
watches/remediates `TARGET_NAMESPACE`, so it has no access outside it:

- `pods`: get, list, watch, delete
- `replicasets.apps`: get (to resolve ReplicaSet → Deployment ownership)

## Testing

Unit tests (fake clientset, no real cluster) cover:

- health detection for every failure reason (`internal/detector`)
- owner resolution: bare pod, Job, StatefulSet, ReplicaSet→Deployment, bare
  ReplicaSet (`internal/remediation`)
- backoff math and the retry cap, including that different workloads have
  independent counters (`internal/remediation`)

```
go test ./...
```

## Failure scenarios (verified)

All of these were run against a live `docker-desktop` Kubernetes cluster,
not just asserted in unit tests:

- **CrashLoopBackOff** — detected, 3 remediation attempts at 2s/4s/8s, then
  stopped. ✅
- **Retry cap survives pod replacement** — same Deployment key held its
  counter across 3 different pod names. ✅
- **Bare/unowned pod** — refused with a log line, not deleted. ✅
- **Owner lookup failure** (RS already gone) — logged as an error, no delete
  attempted — fails safe. ✅
- **Graceful shutdown** — live `SIGTERM` mid-run produced "draining
  in-flight remediations" then "shutdown complete" in order, before the pod
  was removed. ✅
- **Concurrency races** — found and fixed live (see above), then reverified
  clean. ✅
- **Watch reconnect** — implemented (`internal/watcher`), not exercised
  against a real API server outage in this pass; code path handles both a
  `Watch()` error and a closed `ResultChan()`.

## Run locally

```
go run ./cmd/auto-healer
```

Uses `~/.kube/config` (falls back automatically from in-cluster config, which
won't be present on a dev machine). Metrics at `http://localhost:8080/metrics`.

## Deploy to Kubernetes

```
docker build -t pod-auto-healer:latest .
kind load docker-image pod-auto-healer:latest   # only if using kind; not
                                                 # needed on docker-desktop,
                                                 # which shares the local
                                                 # docker daemon

kubectl apply -f deploy/serviceaccount.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
```

Config is env-driven (see `deploy/deployment.yaml`):

| Var | Default |
|---|---|
| `TARGET_NAMESPACE` | `auto-healer-demo` |
| `MAX_REMEDIATION_ATTEMPTS` | `3` |
| `INITIAL_BACKOFF` | `2s` |
| `RESTART_THRESHOLD` | `3` |
| `RETRY_TTL` | `10m` |
| `METRICS_ADDR` | `:8080` |

## Deliberately out of scope

- **Leader election** — `deploy/deployment.yaml` runs `replicas: 1`, which
  sidesteps the split-brain problem leader election solves. Add
  `client-go/tools/leaderelection` before running more than one replica.
- **Integration/e2e test harness** (envtest, kind-in-CI) — unit tests with the
  fake clientset cover the logic; the live-cluster verification above serves
  as the E2E pass. Wiring a real API server into CI is a separate effort not
  needed for this project's scope.
