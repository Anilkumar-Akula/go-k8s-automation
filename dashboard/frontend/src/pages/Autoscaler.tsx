import { useState } from "react";
import { scaleDeployment } from "../api/actions";
import { apiGet } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { AutoscalerDetail } from "../types/dashboard";

export default function Autoscaler() {
  const { data, error, loading } = usePolling<AutoscalerDetail>(() => apiGet("/api/v1/autoscaler"));
  const [replicas, setReplicas] = useState("");
  const [confirming, setConfirming] = useState(false);

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const target = Number(replicas);
  const canScale = replicas !== "" && Number.isInteger(target) && target >= 1;

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

      <h2>Set replica count</h2>
      <div className="action-bar">
        <input
          type="number"
          min={1}
          value={replicas}
          onChange={(e) => setReplicas(e.target.value)}
          placeholder={String(data.currentReplicas)}
          style={{ width: 80, marginRight: 8 }}
        />
        <button className="btn-secondary" disabled={!canScale} onClick={() => setConfirming(true)}>
          Scale
        </button>
      </div>

      <h2>Scaling history</h2>
      <EventList source="autoscaler" />

      {confirming && (
        <ConfirmDialog
          title="Confirm Scaling"
          fields={[
            { label: "Namespace", value: data.namespace },
            { label: "Deployment", value: data.deployment },
            { label: "Current replicas", value: String(data.currentReplicas) },
            { label: "New replicas", value: String(target) },
          ]}
          execute={(reason) => scaleDeployment(data.namespace, data.deployment, target, reason)}
          onClose={() => setConfirming(false)}
        />
      )}
    </section>
  );
}
