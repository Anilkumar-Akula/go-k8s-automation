import { apiGet } from "../api/client";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { HealerDetail } from "../types/dashboard";

export default function AutoHealer() {
  const { data, error, loading } = usePolling<HealerDetail>(() => apiGet("/api/auto-healer"));

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const reasons = Object.entries(data.unhealthyByReason ?? {});

  return (
    <section>
      <h1>Pod Auto-Healer</h1>

      <div className="stat-grid">
        <StatCard label="Pods checked" value={data.podsChecked} />
        <StatCard label="Remediations" value={data.remediationTotal} />
        <StatCard label="Remediation failures" value={data.remediationFailures} />
        <StatCard label="Watch reconnects" value={data.watchReconnects} />
      </div>

      <h2>Unhealthy pods by reason</h2>
      {reasons.length === 0 ? (
        <p className="placeholder-note">No unhealthy pods detected.</p>
      ) : (
        <div className="stat-grid">
          {reasons.map(([reason, count]) => (
            <StatCard key={reason} label={reason} value={count} />
          ))}
        </div>
      )}

      <h2>Recent healing events</h2>
      <EventList source="auto-healer" />
    </section>
  );
}
