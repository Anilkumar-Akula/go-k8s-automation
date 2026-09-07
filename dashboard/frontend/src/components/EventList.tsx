import { useEventStream } from "../hooks/useEventStream";
import type { DashboardEvent } from "../types/dashboard";

interface EventListProps {
  source?: DashboardEvent["source"];
  limit?: number;
}

export default function EventList({ source, limit = 20 }: EventListProps) {
  const { events, error, loading } = useEventStream(limit, source);

  if (loading) return <p className="placeholder-note">Loading...</p>;
  if (error) return <p className="placeholder-note">Failed to load events: {error}</p>;
  if (events.length === 0) return <p className="placeholder-note">No events yet.</p>;

  return (
    <ul className="event-list">
      {events.map((e) => (
        <li key={e.seq}>
          <span className="event-time">{e.time}</span>
          {!source && <span className="event-source">{e.source}</span>}
          <span className="event-target">{e.target}</span>
          <span className="event-message">{e.message}</span>
        </li>
      ))}
    </ul>
  );
}
