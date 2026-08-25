# k8s-pod-auto-healer

Watches Pods in a target namespace, detects CrashLoopBackOff / ImagePullBackOff /
OOMKilled / Failed, and remediates by deleting the pod (its controller replaces
it) — capped at `MAX_REMEDIATION_ATTEMPTS` per owning workload with exponential
backoff.

## Layout

```
cmd/auto-healer/     entrypoint, wiring, graceful shutdown
internal/config/      env-based config
internal/kclient/     in-cluster / kubeconfig client
internal/watcher/      self-reconnecting pod watch
internal/detector/     health checks
internal/remediation/  owner resolution, retry+backoff, delete
internal/metrics/      Prometheus metrics
deploy/                 ServiceAccount, Role/RoleBinding, Deployment
```

## Config (env vars)

| Var | Default |
|---|---|
| `TARGET_NAMESPACE` | `auto-healer-demo` |
| `MAX_REMEDIATION_ATTEMPTS` | `3` |
| `INITIAL_BACKOFF` | `2s` |
| `RESTART_THRESHOLD` | `3` |
| `RETRY_TTL` | `10m` |
| `METRICS_ADDR` | `:8080` |

## Run locally

```
go run ./cmd/auto-healer
```

Uses `~/.kube/config`. Metrics at `http://localhost:8080/metrics`.

## Build & deploy

```
docker build -t pod-auto-healer:latest .
kind load docker-image pod-auto-healer:latest   # if using kind

kubectl apply -f deploy/serviceaccount.yaml
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
```

## Tests

```
go test ./...
```

## Manual failure checklist

- **CrashLoopBackOff**: deploy a crashing pod in the target namespace, confirm 3
  remediation attempts with growing backoff (2s/4s/8s), then `MAX REMEDIATION
  ATTEMPTS REACHED` and no further deletes.
- **Bare pod**: `kubectl run` a pod with no owner in the target namespace,
  crash it, confirm the healer logs "refusing to remediate" and does not
  delete it.
- **Watch interruption**: `kubectl delete apiservice` isn't safe to test, but
  killing the watch client-side (e.g. brief `kubectl proxy` restart, or a
  `kind` control-plane restart) should produce a "watch channel closed,
  reconnecting" log and pod events keep flowing afterward.
- **Graceful shutdown**: send `SIGTERM` (`kill <pid>` or `kubectl delete pod`
  on the healer itself) mid-remediation, confirm it logs "draining in-flight
  remediations" and exits cleanly instead of being killed mid-delete.
- **Healthy pod untouched**: confirm a normally running pod never triggers a
  delete.
- **Multiple pods**: crash two different Deployments concurrently, confirm
  each has its own independent retry counter (`Deployment/<ns>/<name>` key).

## Deliberately out of scope

- **Leader election** — `deploy/deployment.yaml` runs `replicas: 1`. Add
  `client-go/tools/leaderelection` before running more than one replica.
- **Integration/e2e test harness** (envtest, kind-in-CI) — unit tests with the
  fake clientset cover the logic; wiring up a real API server for CI is a
  separate effort not needed for this project's scope.
