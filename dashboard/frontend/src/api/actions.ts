import { apiPost } from "./client";
import type { ActionResult } from "../types/dashboard";

export function restartPod(namespace: string, pod: string, reason: string) {
  return apiPost<ActionResult>("/api/v1/actions/healer/restart", { namespace, pod, reason });
}

export function rollbackDeployment(namespace: string, deployment: string, reason: string) {
  return apiPost<ActionResult>("/api/v1/actions/rollout/rollback", { namespace, deployment, reason });
}

export function applyRecommendation(namespace: string, pod: string, container: string, reason: string) {
  return apiPost<ActionResult>("/api/v1/actions/optimizer/apply", { namespace, pod, container, reason });
}

export function scaleDeployment(namespace: string, deployment: string, replicas: number, reason: string) {
  return apiPost<ActionResult>("/api/v1/actions/autoscaler/scale", { namespace, deployment, replicas, reason });
}
