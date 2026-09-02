import { useEffect, useState } from "react";

interface PollState<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

/** Fetches immediately, then re-fetches every intervalMs until unmounted. */
export function usePolling<T>(fetcher: () => Promise<T>, intervalMs = 10_000): PollState<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function tick() {
      try {
        const result = await fetcher();
        if (!cancelled) {
          setData(result);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    tick();
    const id = setInterval(tick, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
    // fetcher is intentionally excluded: callers pass an inline closure,
    // and re-running the effect on every render would defeat the interval.
  }, [intervalMs]);

  return { data, error, loading };
}
