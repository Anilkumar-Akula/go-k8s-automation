# Unified Kubernetes Automation Dashboard

Real-time, cloud-native control plane dashboard bringing together the **Go + Kubernetes Automation Suite (Projects 01 to 04)**.

```
                          ┌────────────────────────────────┐
                          │   Kubernetes Cluster / API     │
                          │ (Pods, Deployments, RS, Metrics)│
                          └───────────────┬────────────────┘
                                          │ client-go & metrics.k8s.io
                                          ▼
                         ┌──────────────────────────────────┐
                         │   Go Backend Server (dashboard/) │
                         │  - K8s Watchers & Metric Poller │
                         │  - Automation State Aggregator   │
                         │  - REST API & Real-time SSE/WS   │
                         │  - Mock/Demo Mode Fallback       │
                         └────────────────┬─────────────────┘
                                          │ REST + Server-Sent Events (SSE)
                                          ▼
                         ┌──────────────────────────────────┐
                         │   Modern Web Dashboard (UI)      │
                         │  - Overview Control Center       │
                         │  - 01 Auto-Healer Monitor        │
                         │  - 02 Rollout Manager View       │
                         │  - 03 Resource Optimizer View    │
                         │  - 04 Horizontal Autoscaler View │
                         │  - Live Event Stream & Logs      │
                         └──────────────────────────────────┘
```

## Features

- **Overview Radar**: Aggregated cluster metrics, active controller states, and a real-time Server-Sent Events (SSE) activity stream.
- **01 — Pod Auto-Healer**: Real-time pod health evaluation, failure reason badges (`CrashLoopBackOff`, `ImagePullBackOff`, `OOMKilled`, `Failed`), owner resolution hierarchy, and 1-click remediation trigger.
- **02 — Rollout Manager**: Deployment rollout progress tracking, `ProgressDeadlineExceeded` stuck detection, revision history diff, and zero-downtime automated rollback action.
- **03 — Resource Optimizer**: Container CPU & memory usage profiling vs configured requests/limits ($p50$ requests, $p90$ limits), drift analysis, and interactive YAML patch preview & apply modal.
- **04 — Horizontal Autoscaler**: Real-time radial CPU utilization gauge, target vs desired replica counters, asymmetric cooldown countdowns (fast scale up, stabilized scale down).
- **Dual Operating Modes**:
  - **Live Kubernetes Mode**: Automatically connects to your local `~/.kube/config` or in-cluster ServiceAccount.
  - **Simulation / Demo Mode**: Built-in mock data emitter for standalone presentation when offline or presenting without an active cluster.

## Run Locally

```bash
cd dashboard
go mod tidy
go run ./cmd/server
```

Open your browser at `http://localhost:8080`.

## Run Tests

```bash
cd dashboard
go test -v ./...
```

## Deploy to Kubernetes

```bash
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
kubectl port-forward svc/k8s-automation-dashboard 8080:8080
```
