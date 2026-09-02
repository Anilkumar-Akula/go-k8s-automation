import { apiGet } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import type { DashboardEvent } from "../types/dashboard";

interface EventListProps {
  source?: DashboardEvent["source"];
  limit?: number;
}

export default function EventList({ source, limit = 20 }: EventListProps) {
  const params = new URLSearchParams({ limit: String(limit) });
  if (source) params.set("source", source);

  const { data, error, loading } = usePolling<DashboardEvent[]>(() =>
    apiGet(`/api/v1/events?${params.toString()}`),
  );

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load events: {error}</p>;
  if (!data || data.length === 0) return <p className="placeholder-note">No events yet.</p>;

  return (
    <ul className="event-list">
      {data.map((e, i) => (
        <li key={i}>
          <span className="event-time">{e.time}</span>
          {!source && <span className="event-source">{e.source}</span>}
          <span className="event-target">{e.target}</span>
          <span className="event-message">{e.message}</span>
        </li>
      ))}
    </ul>
  );
}
