import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { toApiError } from "./auth";

type QueryState<T> = {
  data: T | null;
  error: ApiError | null;
  loading: boolean;
  unauthorized: boolean;
  forbidden: boolean;
  refetch: () => void;
};

export function useApiQuery<T>(
  key: string,
  fetcher: () => Promise<T>,
  enabled = true
): QueryState<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(enabled);
  const [tick, setTick] = useState(0);

  const refetch = useCallback(() => setTick((n) => n + 1), []);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      setData(null);
      setError(null);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    fetcher()
      .then((result) => {
        if (!cancelled) setData(result);
      })
      .catch((err) => {
        if (!cancelled) {
          setData(null);
          setError(toApiError(err));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [key, enabled, tick]);

  return {
    data,
    error,
    loading,
    unauthorized: error?.status === 401,
    forbidden: error?.status === 403,
    refetch,
  };
}
