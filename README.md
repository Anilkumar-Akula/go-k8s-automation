# Go + Kubernetes Automation Labs

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Automation-326CE5?logo=kubernetes&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

A hands-on collection of Go + Kubernetes automation projects — controllers,
health monitoring, self-healing workloads, and cloud-native backend patterns.
Each project is a standalone Go module in its own directory.

## Projects

| # | Project | Description | Status |
|---|---|---|---|
| 01 | [Pod Auto-Healer](./01-pod-auto-healer) | Detect unhealthy Pods, remediate with owner-aware retry + exponential backoff | ✅ Complete |
| 02 | [Deployment Rollout Manager](./02-rollout-manager) | Automated rollout monitoring and rollback | ✅ Complete |
| 03 | [Kubernetes Resource Optimizer](./03-resource-optimizer) | Analyze CPU/memory usage, recommend request/limit changes | ✅ Complete |
| 04 | [Auto-Scaling Controller](./04-autoscaler) | Custom scaling driven by application metrics | ✅ Complete |
| 05 | Kubernetes Cost Optimizer | Find over-provisioned workloads, reduce resource waste | 🔜 Planned |
| 06 | Canary Deployment Controller | Gradual traffic shift with automatic rollback on errors | 🔜 Planned |
| 07 | PostgreSQL Kubernetes Operator | Operator managing DB lifecycle, backup, failover | 🔜 Planned |
| 08 | Self-Healing Microservice Platform | Centralized detect → diagnose → remediate | 🔜 Planned |
| 09 | GitOps Deployment Engine | Git commit → validate → deploy → health-check → rollback | 🔜 Planned |
| 10 | Custom Kubernetes Controller | Controller/CRD built from scratch on client-go / controller-runtime | 🔜 Planned |

## Tech stack

**Backend** — Go, client-go, goroutines/channels/context, concurrency patterns
**Kubernetes** — Pods, Deployments, ReplicaSets, RBAC, Watch API, controllers
**Infra** — Docker, Kind, docker-desktop Kubernetes, kubectl
**Observability** — Prometheus metrics, structured logging (`log/slog`)

## Repository structure

```
go-k8s-automation/
├── 01-pod-auto-healer/
│   ├── cmd/
│   ├── internal/
│   ├── deploy/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
├── 02-rollout-manager/
│   ├── cmd/
│   ├── internal/
│   ├── deploy/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
├── 03-resource-optimizer/
│   ├── cmd/
│   ├── internal/
│   ├── deploy/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
├── 04-autoscaler/
│   ├── cmd/
│   ├── internal/
│   ├── deploy/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
├── 05-cost-optimizer/        (planned)
├── 06-canary-controller/     (planned)
├── 07-postgres-operator/     (planned)
├── 08-self-healing-platform/ (planned)
├── 09-gitops-engine/         (planned)
├── 10-custom-controller/     (planned)
└── README.md
```

Each project keeps its own `go.mod` — no shared `go.work`, since nothing here
imports across projects. Add one if that changes.

## Project 01 — Kubernetes Pod Auto-Healer

Full detail in [`01-pod-auto-healer/README.md`](./01-pod-auto-healer). Summary:

```
Kubernetes API
      │
 self-reconnecting watch
      │
 health detector (CrashLoopBackOff / ImagePullBackOff / OOMKilled / Failed)
      │
 in-flight dedup guard (1 goroutine per pod)
      │
 owner resolution (Pod → ReplicaSet → Deployment | StatefulSet; refuses bare/Job/DaemonSet)
      │
 retrier (per-workload counter, exponential backoff, TTL cleanup)
      │
 delete pod
```

Demonstrates: client-go watch/informer patterns, Go concurrency (found and
fixed two real races during live testing — duplicate remediation goroutines,
and a pod-termination-window race), graceful shutdown, Prometheus metrics,
least-privilege RBAC, and end-to-end verification against a live cluster —
not just unit tests.

## Project 02 — Deployment Rollout Manager

Full detail in [`02-rollout-manager/README.md`](./02-rollout-manager). Summary:

```
Kubernetes API
      │
 self-reconnecting watch (Deployments)
      │
 stuck-rollout detector (Progressing condition == False, ProgressDeadlineExceeded)
      │
 in-flight dedup guard (1 goroutine per Deployment)
      │
 previous-revision lookup (owned ReplicaSets, revision annotation)
      │
 per-workload retrier (exponential backoff, TTL cleanup)
      │
 patch Deployment.spec.template back to the previous ReplicaSet's template
```

Demonstrates: reusing native Kubernetes signals instead of reimplementing
them (the Deployment controller's own `Progressing`/`ProgressDeadlineExceeded`
condition and revision-annotated ReplicaSet history — the same two things
`kubectl rollout status` and `kubectl rollout undo` read), Prometheus
metrics, least-privilege RBAC, and the same watch/retry/backoff
concurrency pattern as project 01.

## Project 03 — Kubernetes Resource Optimizer

Full detail in [`03-resource-optimizer/README.md`](./03-resource-optimizer). Summary:

```
Kubernetes API (Pod specs) + metrics.k8s.io (PodMetrics)
      │
 sampler joins usage to configured requests/limits by pod/container
      │
 per-container rolling window of usage samples
      │
 percentile recommender (p50 -> request, p90 -> limit)
      │
 drift check vs. configured request/limit
      │
 Prometheus gauges + structured log warnings (no patching)
```

Demonstrates: the `metrics.k8s.io` client (separate from the core
client-go clientset used in projects 01/02), a VPA-style percentile
recommender instead of a fixed usage-plus-margin rule, and a
read-only design — this project only recommends, since patching live
requests/limits can restart Pods and shouldn't happen without an
explicit, separate authorization step.

## Project 04 — Auto-Scaling Controller

Full detail in [`04-autoscaler/README.md`](./04-autoscaler). Summary:

```
Kubernetes API (Deployment + Pods) + metrics.k8s.io (PodMetrics)
      │
 usage.Collect: avg CPU usage / CPU requests, by label selector
      │
 scaler.Decide: ceil(current * observed/target), tolerance band, min/max clamp
      │
 per-direction cooldown (scale up fast, down slow)
      │
 Deployments().UpdateScale (scale subresource only, not the Deployment)
```

Demonstrates: the same ratio-based scaling formula the built-in
HorizontalPodAutoscaler uses, writing only the `deployments/scale`
subresource (least privilege — never touches the Pod template),
asymmetric cooldowns to prevent flapping, and end-to-end verification
against a live cluster with a CPU-stressed demo workload.

## Roadmap

**Phase 1 — Kubernetes fundamentals** *(done, project 01)*
- [x] Connect Go app to Kubernetes (in-cluster + kubeconfig fallback)
- [x] Watch Pods with auto-reconnect
- [x] Detect unhealthy Pods
- [x] Owner-aware remediation
- [x] Retry with exponential backoff
- [x] Graceful shutdown
- [x] Structured logging

**Phase 2 — Production readiness** *(partially done, project 01)*
- [x] Exponential backoff + retry cap
- [x] Prometheus metrics
- [x] Kubernetes RBAC (least privilege)
- [x] Env-based config
- [ ] Leader election (deferred — project 01 runs `replicas: 1`, revisit if it scales out)
- [ ] Rate limiting

**Phase 3 — Advanced automation**
- [x] Deployment/rollout controller (project 02)
- [x] Automated rollback (project 02)
- [ ] Custom Resource Definition
- [ ] Kubernetes Operator

**Phase 4 — Cloud native**
- [ ] GitHub Actions CI
- [ ] Docker image publishing
- [ ] Helm charts
- [ ] AWS EKS deployment

No CI/CD badges yet — added once an actual pipeline exists, not before.

## License

MIT — see [LICENSE](./LICENSE).

## Author

Anil Kumar Akula — Backend Software Engineer | Go | Java | Kubernetes | Distributed Systems
