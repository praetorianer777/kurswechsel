import { Link, useParams } from "react-router";
import { api } from "../api/client";
import { PageHeading } from "../components/PageHeading";
import { StanceLegend } from "../components/StanceBadge";
import { ErrorMessage, Loading } from "../components/Status";
import { TimelineItem } from "../components/TimelineItem";
import { useApi } from "../hooks/useApi";
import { de } from "../i18n/de";

export function Timeline() {
  const { id = "", topic = "" } = useParams();
  const state = useApi((s) => api.timeline(id, topic, s), [id, topic]);

  if (state.status === "loading") return <Loading />;
  if (state.status === "error") return <ErrorMessage error={state.error} />;
  const { politician, topic: t, entries, changes } = state.data;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <PageHeading title={de.timeline.title(politician.name, t.name)}>
          <Link
            to={`/person/${politician.id}`}
            className="underline underline-offset-4"
          >
            {politician.name}
          </Link>
          : {t.name}
        </PageHeading>
        <p className="text-lg">{t.question}</p>
        <p className="font-semibold">
          {de.timeline.summary(entries.length, changes)}
        </p>
      </div>

      <p className="rounded-md border border-line bg-surface px-4 py-3 text-sm">
        {de.timeline.disclaimer}{" "}
        <Link to="/methodik" className="underline underline-offset-4">
          {de.timeline.methodLink}
        </Link>
      </p>

      <StanceLegend />

      {entries.length === 0 ? (
        <p>{de.timeline.empty}</p>
      ) : (
        <section aria-labelledby="aussagen">
          <h2 id="aussagen" className="sr-only">
            {de.timeline.listLabel}
          </h2>
          <ol className="flex flex-col">
            {entries.map((e) => (
              <TimelineItem key={e.paragraph_id} entry={e} topic={t.slug} />
            ))}
          </ol>
        </section>
      )}
    </div>
  );
}
