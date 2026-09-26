import type { ApiError } from "../api/client";
import { de } from "../i18n/de";

export function Loading() {
  return (
    <p role="status" className="py-6 text-muted">
      {de.loading}
    </p>
  );
}

export function ErrorMessage({ error }: { error: ApiError }) {
  return (
    <p
      role="alert"
      className="rounded-md border border-danger bg-danger-soft px-4 py-3 text-ink"
    >
      {error.message}
    </p>
  );
}
