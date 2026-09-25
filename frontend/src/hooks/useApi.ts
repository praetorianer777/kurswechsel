import { useEffect, useState } from "react";
import { ApiError } from "../api/client";

export type ApiState<T> =
  | { status: "loading" }
  | { status: "error"; error: ApiError }
  | { status: "ok"; data: T };

/**
 * Loads data whenever deps change. A result is only shown for the deps it was
 * loaded with, so a slow earlier response can never replace a newer one, and
 * changing deps shows "loading" at once.
 */
export function useApi<T>(
  load: (signal: AbortSignal) => Promise<T>,
  deps: readonly unknown[],
): ApiState<T> {
  const key = JSON.stringify(deps);
  const [result, setResult] = useState<{ key: string; state: ApiState<T> }>();
  useEffect(() => {
    const ctrl = new AbortController();
    load(ctrl.signal).then(
      (data) => {
        if (!ctrl.signal.aborted)
          setResult({ key, state: { status: "ok", data } });
      },
      (err: unknown) => {
        if (ctrl.signal.aborted) return;
        setResult({
          key,
          state: {
            status: "error",
            error: err instanceof ApiError ? err : new ApiError("internal", 0),
          },
        });
      },
    );
    return () => ctrl.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- key covers the caller's deps
  }, [key]);
  return result?.key === key ? result.state : { status: "loading" };
}
