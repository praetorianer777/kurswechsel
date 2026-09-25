import { Link, useParams } from "react-router";
import { api } from "../api/client";
import type { FactionSpan } from "../api/types";
import { PageHeading } from "../components/PageHeading";
import { ErrorMessage, Loading } from "../components/Status";
import { useApi } from "../hooks/useApi";
import { de } from "../i18n/de";
import { formatDate } from "../lib/format";

function span(f: FactionSpan): string {
  if (f.from && f.to)
    return de.person.range(formatDate(f.from), formatDate(f.to));
  if (f.from) return de.person.since(formatDate(f.from));
  return de.person.period(f.period);
}

export function Person() {
  const { id = "" } = useParams();
  const state = useApi((s) => api.politician(id, s), [id]);

  if (state.status === "loading") return <Loading />;
  if (state.status === "error") return <ErrorMessage error={state.error} />;
  const p = state.data;

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col gap-2">
        <PageHeading title={p.name} />
        {p.party && (
          <p className="text-muted">
            {de.person.party}: {p.party}
          </p>
        )}
      </div>

      <section aria-labelledby="themen" className="flex flex-col gap-3">
        <h2 id="themen" className="text-xl font-bold">
          {de.person.topicsHeading}
        </h2>
        {p.topics.length === 0 ? (
          <p>{de.person.noTopics}</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {p.topics.map((t) => (
              <li key={t.slug}>
                <Link
                  to={`/person/${p.id}/${t.slug}`}
                  className="inline-flex min-h-11 items-center gap-2 rounded-md border border-line bg-surface px-4 py-2 hover:bg-line/40"
                >
                  <span className="font-semibold underline underline-offset-4">
                    {t.name}
                  </span>
                  <span className="text-sm text-muted">
                    {de.home.statements(t.entries)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="fraktionen" className="flex flex-col gap-3">
        <h2 id="fraktionen" className="text-xl font-bold">
          {de.person.factions}
        </h2>
        {!p.is_mdb || p.factions.length === 0 ? (
          <p>{de.person.notMdB}</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {p.factions.map((f) => (
              <li key={`${f.period}-${f.faction}-${f.from ?? ""}`}>
                <span className="font-semibold">{f.faction}</span>{" "}
                <span className="text-muted">
                  ({de.person.period(f.period)}, {span(f)})
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
