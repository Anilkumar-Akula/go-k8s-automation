import { useEffect, useRef, useState } from "react";
import { apiGet, API_BASE, getToken } from "../api/client";
import type { DashboardEvent } from "../types/dashboard";

/**
 * Loads the most recent events once, then keeps them current by
 * subscribing to /api/v1/events/stream (SSE) instead of re-polling.
 * Filtering by source happens client-side since the stream carries
 * every event; that's fine at dashboard event volumes.
 */
export function useEventStream(limit: number, source?: DashboardEvent["source"]): {
  events: DashboardEvent[];
  error: string | null;
  loading: boolean;
} {
  const [events, setEvents] = useState<DashboardEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const eventsRef = useRef(events);
  eventsRef.current = events;

  useEffect(() => {
    let cancelled = false;
    const params = new URLSearchParams({ limit: String(limit) });
    if (source) params.set("source", source);

    apiGet<DashboardEvent[]>(`/api/v1/events?${params.toString()}`)
      .then((initial) => {
        if (!cancelled) setEvents(initial);
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    const token = getToken();
    const streamUrl = `${API_BASE}/api/v1/events/stream${token ? `?token=${encodeURIComponent(token)}` : ""}`;
    const es = new EventSource(streamUrl);
    es.onmessage = (msg) => {
      const e = JSON.parse(msg.data) as DashboardEvent;
      if (source && e.source !== source) return;
      setEvents([e, ...eventsRef.current].slice(0, limit));
    };

    return () => {
      cancelled = true;
      es.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [limit, source]);

  return { events, error, loading };
}
