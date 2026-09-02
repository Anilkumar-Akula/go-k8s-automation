import { useState } from "react";
import { restartPod } from "../api/actions";
import { apiGet } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import type { HealerDetail, UnhealthyPod } from "../types/dashboard";

export default function AutoHealer() {
  const { data, error, loading } = usePolling<HealerDetail>(() => apiGet("/api/v1/auto-healer"));
  const pods = usePolling<UnhealthyPod[]>(() => apiGet("/api/v1/auto-healer/pods"), 10_000);
  const [target, setTarget] = useState<UnhealthyPod | null>(null);

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const reasons = Object.entries(data.unhealthyByReason ?? {});
  const unhealthyPods = pods.data ?? [];

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

      <h2>Currently unhealthy pods</h2>
      {unhealthyPods.length === 0 ? (
        <p className="placeholder-note">None right now.</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Pod</th>
                <th>Reason</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {unhealthyPods.map((p) => (
                <tr key={`${p.namespace}/${p.pod}`}>
                  <td>
                    {p.namespace}/{p.pod}
                  </td>
                  <td>{p.reason}</td>
                  <td>
                    <button className="btn-secondary" onClick={() => setTarget(p)}>
                      Restart
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2>Recent healing events</h2>
      <EventList source="auto-healer" />

      {target && (
        <ConfirmDialog
          title="Restart Pod"
          fields={[
            { label: "Namespace", value: target.namespace },
            { label: "Pod", value: target.pod },
            { label: "Reason detected", value: target.reason },
          ]}
          execute={(reason) => restartPod(target.namespace, target.pod, reason)}
          onClose={() => setTarget(null)}
        />
      )}
    </section>
  );
}
