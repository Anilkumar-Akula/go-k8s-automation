import { apiGet } from "../api/client";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { RolloutDetail } from "../types/dashboard";

export default function RolloutManager() {
  const { data, error, loading } = usePolling<RolloutDetail>(() => apiGet("/api/rollout-manager"));

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const reasons = Object.entries(data.stuckByReason ?? {});

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

      <h2>Recent rollback events</h2>
      <EventList source="rollout-manager" />
    </section>
  );
}
