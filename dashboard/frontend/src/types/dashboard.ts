export interface ClusterSummary {
  nodeCount: number;
  nodesReady: number;
  podsTotal: number;
  podsRunning: number;
  podsPending: number;
  podsFailed: number;
}

export interface DashboardEvent {
  seq: number;
  time: string;
  source: "auto-healer" | "rollout-manager" | "resource-optimizer" | "autoscaler";
  target: string;
  message: string;
}

interface AutomationBase {
  name: string;
  namespace: string;
}

export interface AutoHealerSummary extends AutomationBase {
  name: "auto-healer";
  podsChecked: number;
  remediations: number;
}

export interface RolloutManagerSummary extends AutomationBase {
  name: "rollout-manager";
  deploymentsChecked: number;
  rollbacks: number;
}

export interface ResourceOptimizerSummary extends AutomationBase {
  name: "resource-optimizer";
  containersTracked: number;
  driftCount: number;
}

export interface AutoscalerSummary extends AutomationBase {
  name: "autoscaler";
  currentReplicas: number;
  desiredReplicas: number;
}

export type AutomationSummary =
  | AutoHealerSummary
  | RolloutManagerSummary
  | ResourceOptimizerSummary
  | AutoscalerSummary;

export interface Overview {
  cluster: ClusterSummary;
  lastPoll: string;
  automations: AutomationSummary[];
  recentEvents: DashboardEvent[];
}

export interface HealerDetail {
  podsChecked: number;
  remediationTotal: number;
  remediationFailures: number;
  watchReconnects: number;
  unhealthyByReason: Record<string, number>;
}

export interface RolloutDetail {
  deploymentsChecked: number;
  rollbackTotal: number;
  rollbackFailures: number;
  watchReconnects: number;
  stuckByReason: Record<string, number>;
}

export interface Recommendation {
  namespace: string;
  pod: string;
  container: string;
  reqCpuMilli: number;
  limCpuMilli: number;
  reqMemBytes: number;
  limMemBytes: number;
}

export interface DriftEntry {
  namespace: string;
  pod: string;
  container: string;
  resource: string;
  field: string;
  direction: "under" | "over";
  count: number;
}

export interface OptimizerDetail {
  containersTracked: number;
  samplesCollected: number;
  recommendations: Recommendation[] | null;
  drift: DriftEntry[] | null;
}

export interface AutoscalerDetail {
  namespace: string;
  deployment: string;
  currentReplicas: number;
  desiredReplicas: number;
  utilizationPercent: number;
  scaleUpEvents: number;
  scaleDownEvents: number;
}

export interface UnhealthyPod {
  namespace: string;
  pod: string;
  reason: string;
}

export interface StuckDeployment {
  namespace: string;
  deployment: string;
  reason: string;
}

export interface AuditEvent {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  project: string;
  namespace: string;
  resource: string;
  oldValue?: string;
  newValue?: string;
  reason: string;
  status: "success" | "failed";
}

export interface ActionResult {
  status: string;
  oldValue?: string;
  newValue?: string;
  audit: AuditEvent;
}
