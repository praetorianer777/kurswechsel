import { useEffect, useId, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { api } from "../api/client";
import type { PoliticianSummary } from "../api/types";
import { ErrorMessage, Loading } from "../components/Status";
import { PageHeading } from "../components/PageHeading";
import { useApi } from "../hooks/useApi";
import { de } from "../i18n/de";

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

export function Home() {
  const [params, setParams] = useSearchParams();
  const q = params.get("q") ?? "";
  const topic = params.get("thema") ?? "";
  const [input, setInput] = useState(q);
  const debounced = useDebounced(input, 250);
  const id = useId();

  useEffect(() => {
    if (debounced === q) return;
    const next = new URLSearchParams(params);
    if (debounced) next.set("q", debounced);
    else next.delete("q");
    setParams(next, { replace: true });
  }, [debounced, q, params, setParams]);

  const topics = useApi((s) => api.topics(s), []);
  const searching = q !== "" || topic !== "";
  const results = useApi(
    (s) => (searching ? api.politicians(q, topic, s) : Promise.resolve([])),
    [q, topic, searching],
  );

  const setTopic = (value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set("thema", value);
    else next.delete("thema");
    setParams(next, { replace: true });
  };

  return (
    <div className="flex flex-col gap-10">
      <section className="flex flex-col gap-3">
        <PageHeading title={de.home.title} />
        <p className="max-w-prose text-lg text-muted">{de.home.intro}</p>
      </section>

      <section aria-labelledby={`${id}-search`} className="flex flex-col gap-4">
        <h2 id={`${id}-search`} className="text-xl font-bold">
          {de.home.searchHeading}
        </h2>
        <form
          role="search"
          onSubmit={(e) => e.preventDefault()}
          className="grid gap-4 sm:grid-cols-[2fr_1fr]"
        >
          <div className="flex flex-col gap-1">
            <label htmlFor={`${id}-q`} className="font-semibold">
              {de.home.searchLabel}
            </label>
            <input
              id={`${id}-q`}
              type="search"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              autoComplete="off"
              aria-describedby={`${id}-hint`}
              className="field"
            />
            <p id={`${id}-hint`} className="text-sm text-muted">
              {de.home.searchHint}
            </p>
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor={`${id}-topic`} className="font-semibold">
              {de.home.topicLabel}
            </label>
            <select
              id={`${id}-topic`}
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              className="field"
            >
              <option value="">{de.home.allTopics}</option>
              {topics.status === "ok" &&
                topics.data.map((t) => (
                  <option key={t.slug} value={t.slug}>
                    {t.name}
                  </option>
                ))}
            </select>
          </div>
        </form>
        <div aria-live="polite" aria-atomic="false">
          {searching && results.status === "loading" && <Loading />}
          {results.status === "error" && <ErrorMessage error={results.error} />}
          {searching && results.status === "ok" && (
            <Results people={results.data} topic={topic} />
          )}
        </div>
      </section>

      <section aria-labelledby={`${id}-topics`} className="flex flex-col gap-4">
        <h2 id={`${id}-topics`} className="text-xl font-bold">
          {de.home.topicsHeading}
        </h2>
        {topics.status === "loading" && <Loading />}
        {topics.status === "error" && <ErrorMessage error={topics.error} />}
        {topics.status === "ok" && (
          <ul className="grid gap-4 sm:grid-cols-2">
            {topics.data.map((t) => (
              <li
                key={t.slug}
                className="rounded-lg border border-line bg-surface p-4"
              >
                <h3 className="text-lg font-semibold">{t.name}</h3>
                <p className="mt-1">{t.question}</p>
                <p className="mt-2 text-sm text-muted">
                  {de.home.topicStats(t.people, t.entries)}
                </p>
                <button
                  type="button"
                  onClick={() => setTopic(t.slug)}
                  className="btn-secondary mt-3"
                >
                  {de.home.topicLabel}: {t.name}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby={`${id}-how`} className="flex flex-col gap-3">
        <h2 id={`${id}-how`} className="text-xl font-bold">
          {de.home.howHeading}
        </h2>
        <ol className="list-decimal space-y-2 pl-6">
          {de.home.how.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
      </section>
    </div>
  );
}

function Results({
  people,
  topic,
}: {
  people: PoliticianSummary[];
  topic: string;
}) {
  return (
    <div className="flex flex-col gap-2">
      <p className="font-semibold">{de.home.results(people.length)}</p>
      {people.length > 0 && (
        <ul className="divide-y divide-line rounded-lg border border-line bg-surface">
          {people.map((p) => (
            <li key={p.id}>
              <Link
                to={topic ? `/person/${p.id}/${topic}` : `/person/${p.id}`}
                className="flex min-h-11 flex-wrap items-baseline justify-between gap-x-4 px-4 py-3 hover:bg-line/40"
              >
                <span className="font-semibold underline underline-offset-4">
                  {p.name}
                </span>
                <span className="text-sm text-muted">
                  {[
                    p.faction || p.role,
                    topic
                      ? de.home.statements(p.entries)
                      : de.home.speeches(p.entries),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
