import { errorMessage } from "../i18n/de";
import type {
  PoliticianDetail,
  PoliticianSummary,
  ReportRequest,
  TimelineResponse,
  TopicSummary,
} from "./types";

/** An API failure with a message that can be shown to visitors as is. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, status: number) {
    super(errorMessage(code));
    this.code = code;
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      ...init,
      headers: { Accept: "application/json", ...init?.headers },
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") throw err;
    throw new ApiError("network", 0);
  }
  if (!res.ok) {
    let code = "internal";
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) code = body.error;
    } catch {
      // The body was not JSON; the generic message applies.
    }
    throw new ApiError(code, res.status);
  }
  return (await res.json()) as T;
}

export const api = {
  topics: (signal?: AbortSignal) =>
    request<TopicSummary[]>("/api/topics", { signal }),
  politicians: (q: string, topic: string, signal?: AbortSignal) => {
    const params = new URLSearchParams();
    if (q) params.set("q", q);
    if (topic) params.set("topic", topic);
    const qs = params.toString();
    return request<PoliticianSummary[]>(
      `/api/politicians${qs ? `?${qs}` : ""}`,
      { signal },
    );
  },
  politician: (id: string, signal?: AbortSignal) =>
    request<PoliticianDetail>(`/api/politicians/${encodeURIComponent(id)}`, {
      signal,
    }),
  timeline: (politician: string, topic: string, signal?: AbortSignal) =>
    request<TimelineResponse>(
      `/api/timeline?${new URLSearchParams({ politician, topic }).toString()}`,
      { signal },
    ),
  report: (body: ReportRequest) =>
    request<{ status: string }>("/api/reports", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
