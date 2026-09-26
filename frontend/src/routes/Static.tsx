import { Link } from "react-router";
import { api } from "../api/client";
import { PageHeading } from "../components/PageHeading";
import { useApi } from "../hooks/useApi";
import { de } from "../i18n/de";
import { formatDate } from "../lib/format";

const percent = new Intl.NumberFormat("de-DE", {
  style: "percent",
  maximumFractionDigits: 0,
});

/** Shows the measured accuracy, but only from a run against a reviewed gold set. */
function Accuracy() {
  const a = de.accuracy;
  const state = useApi((s) => api.evaluation("wehrpflicht", s), []);
  return (
    <section aria-labelledby="genauigkeit" className="flex flex-col gap-2">
      <h2 id="genauigkeit" className="text-xl font-bold">
        {a.heading}
      </h2>
      <p>{a.intro}</p>
      {state.status === "ok" ? (
        <>
          <table className="w-full border-collapse text-left">
            <caption className="mb-2 text-left text-sm text-muted">
              {a.caption("Wehrpflicht", formatDate(state.data.created_at))}
            </caption>
            <thead>
              <tr className="border-b border-line">
                <th scope="col" className="py-2 pr-4">
                  {a.metric}
                </th>
                <th scope="col" className="py-2 text-right">
                  {a.value}
                </th>
              </tr>
            </thead>
            <tbody>
              {(
                [
                  [a.stanceAccuracy, state.data.stance_accuracy],
                  [a.relevanceRecall, state.data.relevance_recall],
                  [a.relevancePrecision, state.data.relevance_precision],
                  [a.flipRate, state.data.flip_rate],
                ] as const
              ).map(([label, value]) => (
                <tr key={label} className="border-b border-line">
                  <th scope="row" className="py-2 pr-4 font-normal">
                    {label}
                  </th>
                  <td className="py-2 text-right font-semibold tabular-nums">
                    {percent.format(value)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="text-sm text-muted">
            {a.basis(state.data.items, state.data.classifier)}
          </p>
        </>
      ) : (
        state.status !== "loading" && <p>{a.pending}</p>
      )}
    </section>
  );
}

export function Methodology() {
  const m = de.methodology;
  return (
    <article className="flex max-w-prose flex-col gap-6">
      <PageHeading title={m.title} />
      <p className="text-lg">{m.intro}</p>
      {m.sections.map((s) => (
        <section key={s.heading} className="flex flex-col gap-2">
          <h2 className="text-xl font-bold">{s.heading}</h2>
          <p>{s.body}</p>
        </section>
      ))}
      <Accuracy />
    </article>
  );
}

export function Imprint() {
  return (
    <article className="flex max-w-prose flex-col gap-4">
      <PageHeading title={de.imprint.title} />
      <p>{de.imprint.body}</p>
    </article>
  );
}

export function Privacy() {
  return (
    <article className="flex max-w-prose flex-col gap-4">
      <PageHeading title={de.privacy.title} />
      <p>{de.privacy.body}</p>
    </article>
  );
}

export function NotFound() {
  return (
    <article className="flex flex-col gap-4">
      <PageHeading title={de.notFound.title} />
      <p>{de.notFound.body}</p>
      <p>
        <Link to="/" className="underline underline-offset-4">
          {de.notFound.back}
        </Link>
      </p>
    </article>
  );
}
