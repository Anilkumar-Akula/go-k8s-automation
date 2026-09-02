import { apiGet } from "../api/client";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { AutomationSummary, Overview as OverviewData } from "../types/dashboard";

const AUTOMATION_LABELS: Record<AutomationSummary["name"], string> = {
  "auto-healer": "Pod Auto-Healer",
  "rollout-manager": "Rollout Manager",
  "resource-optimizer": "Resource Optimizer",
  autoscaler: "Auto-Scaling Controller",
};

function automationHeadline(a: AutomationSummary): string {
  switch (a.name) {
    case "auto-healer":
      return `${a.remediations} remediation(s) · ${a.podsChecked} pods checked`;
    case "rollout-manager":
      return `${a.rollbacks} rollback(s) · ${a.deploymentsChecked} deployments checked`;
    case "resource-optimizer":
      return `${a.driftCount} drift finding(s) · ${a.containersTracked} containers tracked`;
    case "autoscaler":
      return `${a.currentReplicas} / ${a.desiredReplicas} replicas (current/desired)`;
  }
}

export default function Overview() {
  const { data, error, loading } = usePolling<OverviewData>(() => apiGet("/api/overview"));

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load overview: {error}</p>;
  if (!data) return null;

  const { cluster, automations, recentEvents } = data;

  return (
    <section>
      <h1>Overview</h1>

      <div className="stat-grid">
        <StatCard label="Nodes" value={`${cluster.nodesReady}/${cluster.nodeCount}`} />
        <StatCard label="Pods running" value={cluster.podsRunning} />
        <StatCard label="Pods pending" value={cluster.podsPending} />
        <StatCard label="Pods failed" value={cluster.podsFailed} />
      </div>

      <h2>Automations</h2>
      <div className="automation-grid">
        {automations.map((a) => (
          <div className="automation-card" key={a.name}>
            <div className="automation-name">{AUTOMATION_LABELS[a.name]}</div>
            <div className="automation-namespace">{a.namespace}</div>
            <div className="automation-headline">{automationHeadline(a)}</div>
          </div>
        ))}
      </div>

      <h2>Recent events</h2>
      {recentEvents.length === 0 ? (
        <p className="placeholder-note">No events yet.</p>
      ) : (
        <ul className="event-list">
          {recentEvents.map((e, i) => (
            <li key={i}>
              <span className="event-time">{e.time}</span>
              <span className="event-source">{e.source}</span>
              <span className="event-target">{e.target}</span>
              <span className="event-message">{e.message}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
