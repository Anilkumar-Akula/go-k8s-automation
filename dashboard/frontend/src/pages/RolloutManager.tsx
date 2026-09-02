import { useState } from "react";
import { rollbackDeployment } from "../api/actions";
import { apiGet } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { RolloutDetail, StuckDeployment } from "../types/dashboard";

export default function RolloutManager() {
  const { data, error, loading } = usePolling<RolloutDetail>(() => apiGet("/api/v1/rollout-manager"));
  const stuck = usePolling<StuckDeployment[]>(() => apiGet("/api/v1/rollout-manager/deployments"), 10_000);
  const [target, setTarget] = useState<StuckDeployment | null>(null);

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const reasons = Object.entries(data.stuckByReason ?? {});
  const stuckDeployments = stuck.data ?? [];

  return (
    <section>
      <h1>Rollout Manager</h1>

      <div className="stat-grid">
        <StatCard label="Deployments checked" value={data.deploymentsChecked} />
        <StatCard label="Rollbacks" value={data.rollbackTotal} />
        <StatCard label="Rollback failures" value={data.rollbackFailures} />
        <StatCard label="Watch reconnects" value={data.watchReconnects} />
      </div>

      <h2>Stuck rollouts by reason</h2>
      {reasons.length === 0 ? (
        <p className="placeholder-note">No stuck rollouts detected.</p>
      ) : (
        <div className="stat-grid">
          {reasons.map(([reason, count]) => (
            <StatCard key={reason} label={reason} value={count} />
          ))}
        </div>
      )}

      <h2>Currently stuck deployments</h2>
      {stuckDeployments.length === 0 ? (
        <p className="placeholder-note">None right now.</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Deployment</th>
                <th>Reason</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {stuckDeployments.map((d) => (
                <tr key={`${d.namespace}/${d.deployment}`}>
                  <td>
                    {d.namespace}/{d.deployment}
                  </td>
                  <td>{d.reason}</td>
                  <td>
                    <button className="btn-secondary" onClick={() => setTarget(d)}>
                      Rollback
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2>Recent rollback events</h2>
      <EventList source="rollout-manager" />

      {target && (
        <ConfirmDialog
          title="Rollback Deployment"
          fields={[
            { label: "Namespace", value: target.namespace },
            { label: "Deployment", value: target.deployment },
            { label: "Reason detected", value: target.reason },
          ]}
          execute={(reason) => rollbackDeployment(target.namespace, target.deployment, reason)}
          onClose={() => setTarget(null)}
        />
      )}
    </section>
  );
}
