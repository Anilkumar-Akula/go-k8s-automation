# 02 — Deployment Rollout Manager

Watches Deployments in a namespace and automatically rolls back any
Deployment whose rollout gets stuck, instead of waiting for a human to
notice and run `kubectl rollout undo`.

```
Kubernetes API
      │
 self-reconnecting watch (Deployments)
      │
 stuck-rollout detector (Progressing condition == False, reason
 ProgressDeadlineExceeded — the same signal `kubectl rollout status` waits on)
      │
 in-flight dedup guard (1 goroutine per Deployment)
      │
 previous-revision lookup (owned ReplicaSets, deployment.kubernetes.io/revision
 annotation — the same history `kubectl rollout undo` reads)
      │
 per-workload retrier (exponential backoff, attempt cap, TTL cleanup)
      │
 patch Deployment.spec.template back to the previous ReplicaSet's template
```

Because the rolled-back template hash matches the previous ReplicaSet, the
Deployment controller scales that ReplicaSet back up instead of creating a
new one — the exact mechanism `kubectl rollout undo` relies on, reused here
instead of reimplemented.

## Why this approach

- **No custom stuck-rollout timer.** `progressDeadlineSeconds` (default
  600s) is already enforced by the Deployment controller; this project just
  reads the `Progressing` condition it sets.
- **No custom revision history.** ReplicaSets already carry the
  `deployment.kubernetes.io/revision` annotation and are kept around per
  `spec.revisionHistoryLimit` (default 10); rollback just finds the
  previous one and reuses its pod template.
- Rollback attempts are capped and backed off per Deployment
  (`internal/rollback`), so a rollback that itself fails to converge
  doesn't retry forever.

## Run locally

```sh
go run ./cmd/rollout-manager
```

Requires a reachable cluster (`~/.kube/config` or in-cluster). Config is
env-based:

| Env var | Default | Meaning |
|---|---|---|
| `TARGET_NAMESPACE` | `rollout-manager-demo` | Namespace to watch |
| `MAX_ROLLBACK_ATTEMPTS` | `3` | Per-Deployment rollback cap |
| `INITIAL_BACKOFF` | `2s` | First retry delay, doubles each attempt |
| `RETRY_TTL` | `10m` | How long a Deployment's attempt count is remembered |
| `METRICS_ADDR` | `:8080` | Prometheus `/metrics` bind address |

## Deploy

```sh
kubectl apply -f deploy/serviceaccount.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
```

RBAC is a namespaced `Role`, not a `ClusterRole` — the manager only
watches/patches Deployments in `TARGET_NAMESPACE`.

## Demo

Trigger a stuck rollout by pointing a Deployment at a bad image with a
short deadline, and watch this manager roll it back:

```sh
kubectl create deployment demo --image=nginx:1.25 -n rollout-manager-demo
kubectl patch deployment demo -n rollout-manager-demo --type=json \
  -p '[{"op":"add","path":"/spec/progressDeadlineSeconds","value":30}]'
kubectl set image deployment/demo nginx=nginx:does-not-exist -n rollout-manager-demo
```

Within `progressDeadlineSeconds` the Deployment's `Progressing` condition
flips to `ProgressDeadlineExceeded`, and this manager patches the pod
template back to the last working revision.
