import { render } from "@testing-library/react";
import axe from "axe-core";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { expect, vi } from "vitest";
import type { TimelineEntry, TimelineResponse } from "../api/types";

type Route =
  | { status?: number; body: unknown }
  | ((init?: RequestInit) => { status?: number; body: unknown });

/** Replaces fetch with a table of path → response and returns the mock. */
export function mockFetch(routes: Record<string, Route>) {
  const fn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const route = routes[url];
    if (!route)
      return new Response(JSON.stringify({ error: "not_found" }), {
        status: 404,
      });
    const { status = 200, body } =
      typeof route === "function" ? route(init) : route;
    return new Response(JSON.stringify(body), { status });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

export function renderAt(ui: ReactElement, path = "/") {
  return render(<MemoryRouter initialEntries={[path]}>{ui}</MemoryRouter>);
}

/**
 * Runs axe on a rendered tree. Colour contrast needs real layout, so it is
 * checked in the Playwright suite instead.
 */
export async function expectAccessible(container: Element) {
  const results = await axe.run(container, {
    rules: { "color-contrast": { enabled: false }, region: { enabled: false } },
  });
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`,
  );
  expect(summary).toEqual([]);
}

export function entry(overrides: Partial<TimelineEntry> = {}): TimelineEntry {
  return {
    paragraph_id: 1,
    speech_id: "ID1",
    date: "2019-03-01T00:00:00Z",
    period: 19,
    session: 50,
    page: 5500,
    faction: "CDU/CSU",
    text: "Wir müssen die Wehrpflicht wieder einführen.",
    stance: "dafuer",
    quote: "die Wehrpflicht wieder einführen",
    rationale: "Fordert die Wiedereinführung.",
    confidence: 0.9,
    model: "ollama/qwen3:30b-a3b",
    source_url: "https://dserver.bundestag.de/btp/19/19050.pdf",
    change: false,
    ...overrides,
  };
}

export const timelineResponse: TimelineResponse = {
  politician: {
    id: "11000001",
    name: "Jürgen Müller",
    party: "CDU",
    is_mdb: true,
    factions: [],
    topics: [],
  },
  topic: {
    slug: "wehrpflicht",
    name: "Wehrpflicht",
    question: "Soll der Wehrdienst verpflichtend sein?",
  },
  changes: 1,
  entries: [
    entry(),
    entry({
      paragraph_id: 2,
      speech_id: "ID2",
      date: "2024-03-01T00:00:00Z",
      period: 20,
      session: 90,
      text: "Der Wehrdienst muss freiwillig bleiben.",
      stance: "dagegen",
      quote: "freiwillig bleiben",
      change: true,
      previous_stance: "dafuer",
    }),
  ],
};
