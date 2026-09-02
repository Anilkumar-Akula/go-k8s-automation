import { useState } from "react";
import { applyRecommendation } from "../api/actions";
import { apiGet } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import EventList from "../components/EventList";
import StatCard from "../components/StatCard";
import { usePolling } from "../hooks/usePolling";
import { formatBytes, formatMilli } from "../lib/format";
import type { OptimizerDetail, Recommendation } from "../types/dashboard";

export default function ResourceOptimizer() {
  const { data, error, loading } = usePolling<OptimizerDetail>(() =>
    apiGet("/api/v1/resource-optimizer"),
  );
  const [target, setTarget] = useState<Recommendation | null>(null);

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;
  if (!data) return null;

  const recommendations = data.recommendations ?? [];
  const drift = data.drift ?? [];

  return (
    <section>
      <h1>Resource Optimizer</h1>

      <div className="stat-grid">
        <StatCard label="Containers tracked" value={data.containersTracked} />
        <StatCard label="Samples collected" value={data.samplesCollected} />
        <StatCard label="Drift findings" value={drift.length} />
      </div>

      <h2>Recommendations</h2>
      {recommendations.length === 0 ? (
        <p className="placeholder-note">No recommendations yet — waiting on usage samples.</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Pod</th>
                <th>Container</th>
                <th>Req CPU</th>
                <th>Lim CPU</th>
                <th>Req Mem</th>
                <th>Lim Mem</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {recommendations.map((r) => (
                <tr key={`${r.namespace}/${r.pod}/${r.container}`}>
                  <td>
                    {r.namespace}/{r.pod}
                  </td>
                  <td>{r.container}</td>
                  <td>{formatMilli(r.reqCpuMilli)}</td>
                  <td>{formatMilli(r.limCpuMilli)}</td>
                  <td>{formatBytes(r.reqMemBytes)}</td>
                  <td>{formatBytes(r.limMemBytes)}</td>
                  <td>
                    <button className="btn-secondary" onClick={() => setTarget(r)}>
                      Apply
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2>Drift</h2>
      {drift.length === 0 ? (
        <p className="placeholder-note">No drift detected.</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Pod</th>
                <th>Container</th>
                <th>Resource</th>
                <th>Field</th>
                <th>Direction</th>
                <th>Count</th>
              </tr>
            </thead>
            <tbody>
              {drift.map((d) => (
                <tr key={`${d.namespace}/${d.pod}/${d.container}/${d.resource}/${d.field}`}>
                  <td>
                    {d.namespace}/{d.pod}
                  </td>
                  <td>{d.container}</td>
                  <td>{d.resource}</td>
                  <td>{d.field}</td>
                  <td className={`drift-${d.direction}`}>{d.direction}</td>
                  <td>{d.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <h2>Recent events</h2>
      <EventList source="resource-optimizer" />

      {target && (
        <ConfirmDialog
          title="Apply Recommendation"
          fields={[
            { label: "Namespace", value: target.namespace },
            { label: "Pod", value: target.pod },
            { label: "Container", value: target.container },
            { label: "New request", value: `cpu ${formatMilli(target.reqCpuMilli)}, mem ${formatBytes(target.reqMemBytes)}` },
            { label: "New limit", value: `cpu ${formatMilli(target.limCpuMilli)}, mem ${formatBytes(target.limMemBytes)}` },
          ]}
          execute={(reason) => applyRecommendation(target.namespace, target.pod, target.container, reason)}
          onClose={() => setTarget(null)}
        />
      )}
    </section>
  );
}
