import { apiGet } from "../api/client";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { AutoscalerDetail } from "../types/dashboard";

export default function Autoscaler() {
  const { data, error, loading } = usePolling<AutoscalerDetail>(() => apiGet("/api/autoscaler"));

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  return (
    <section>
      <h1>Auto-Scaling Controller</h1>
      <p className="placeholder-note">
        {data.namespace}/{data.deployment}
      </p>

      <div className="stat-grid">
        <StatCard label="Current replicas" value={data.currentReplicas} />
        <StatCard label="Desired replicas" value={data.desiredReplicas} />
        <StatCard label="CPU utilization" value={`${data.utilizationPercent.toFixed(1)}%`} />
        <StatCard label="Scale-up events" value={data.scaleUpEvents} />
        <StatCard label="Scale-down events" value={data.scaleDownEvents} />
      </div>

      <h2>Scaling history</h2>
      <EventList source="autoscaler" />
    </section>
  );
}
