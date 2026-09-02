export interface ClusterSummary {
  nodeCount: number;
  nodesReady: number;
  podsTotal: number;
  podsRunning: number;
  podsPending: number;
  podsFailed: number;
}

export interface DashboardEvent {
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
