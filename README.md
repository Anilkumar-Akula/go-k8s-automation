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
| 01 | [Kubernetes Pod Auto-Healer](./01-pod-auto-healer) | Watches Pods, detects CrashLoopBackOff/OOMKilled/ImagePullBackOff/Failed, remediates with owner-aware retry+backoff | ✅ Complete |
| 02 | Kubernetes Deployment Watcher | Monitor Deployment health and rollout status | 🔜 Planned |
| 03 | Kubernetes Resource Monitor | Track CPU/memory usage against requests/limits | 🔜 Planned |
| 04 | Kubernetes Log Analyzer | Analyze application logs and detect failures | 🔜 Planned |
| 05 | Kubernetes Restart Controller | Detect repeated crashes across workloads | 🔜 Planned |
| 06 | Kubernetes Cost Monitor | Identify resource over-provisioning | 🔜 Planned |

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
├── 02-deployment-watcher/   (planned)
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
- [ ] Deployment/rollout controller (project 02)
- [ ] Custom Resource Definition
- [ ] Kubernetes Operator
- [ ] Automated rollback

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
