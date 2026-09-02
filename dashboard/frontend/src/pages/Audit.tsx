import { apiGet } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import type { AuditEvent } from "../types/dashboard";

export default function Audit() {
  const { data, error, loading } = usePolling<AuditEvent[]>(() => apiGet("/api/v1/audit?limit=100"));

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load: {error}</p>;

  const events = data ?? [];

  return (
    <section>
      <h1>Audit History</h1>
      {events.length === 0 ? (
        <p className="placeholder-note">No control actions taken yet.</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Actor</th>
                <th>Action</th>
                <th>Project</th>
                <th>Resource</th>
                <th>Old</th>
                <th>New</th>
                <th>Reason</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {events.map((e) => (
                <tr key={e.id}>
                  <td>{e.timestamp}</td>
                  <td>{e.actor}</td>
                  <td>{e.action}</td>
                  <td>{e.project}</td>
                  <td>
                    {e.namespace}/{e.resource}
                  </td>
                  <td>{e.oldValue}</td>
                  <td>{e.newValue}</td>
                  <td>{e.reason}</td>
                  <td className={`status-${e.status}`}>{e.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
