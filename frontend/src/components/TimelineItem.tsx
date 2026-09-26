import type { TimelineEntry } from "../api/types";
import { de } from "../i18n/de";
import { formatDate, isoDay } from "../lib/format";
import { Highlighted } from "./Highlighted";
import { ReportDialog } from "./ReportDialog";
import { StanceBadge } from "./StanceBadge";

export function TimelineItem({
  entry,
  topic,
}: {
  entry: TimelineEntry;
  topic: string;
}) {
  const date = formatDate(entry.date);
  const headingId = `eintrag-${entry.paragraph_id}`;
  return (
    <li className="relative pb-8 pl-8 last:pb-0">
      <span
        aria-hidden="true"
        className="absolute top-2 left-[0.4375rem] h-full w-0.5 bg-line"
      />
      <span
        aria-hidden="true"
        className={`absolute top-1.5 left-0 size-4 rounded-full border-2 border-surface ${
          entry.change ? "bg-accent ring-4 ring-accent/30" : "bg-muted"
        }`}
      />
      <article
        aria-labelledby={headingId}
        className={`rounded-lg border bg-surface p-4 shadow-sm ${
          entry.change ? "border-accent" : "border-line"
        }`}
      >
        {entry.change && entry.previous_stance && (
          <p className="mb-3 inline-flex flex-wrap items-center gap-x-2 rounded-md bg-accent-soft px-3 py-1 text-sm font-semibold text-ink">
            <span>{de.timeline.change}:</span>
            <span className="font-normal">
              {de.timeline.changeDetail(
                de.stance[entry.previous_stance],
                de.stance[entry.stance],
              )}
            </span>
          </p>
        )}
        <header className="flex flex-wrap items-center justify-between gap-2">
          <h3 id={headingId} className="font-semibold">
            <time dateTime={isoDay(entry.date)}>{date}</time>
          </h3>
          <StanceBadge stance={entry.stance} />
        </header>
        <blockquote className="mt-3 border-l-4 border-line pl-3 leading-relaxed">
          <p>
            <Highlighted text={entry.text} quote={entry.quote} />
          </p>
        </blockquote>
        {entry.quote && (
          <p className="sr-only">
            {de.timeline.quoteLabel}: {entry.quote}
          </p>
        )}
        <p className="mt-3 text-sm">
          <span className="font-semibold">{de.timeline.rationale}:</span>{" "}
          {entry.rationale}
        </p>
        <footer className="mt-3 flex flex-wrap items-center justify-between gap-2 text-sm text-muted">
          <p>
            {de.timeline.protocol(entry.period, entry.session)}
            {entry.page ? `, ${de.timeline.page(entry.page)}` : ""}
            {entry.faction
              ? ` · ${entry.faction}`
              : entry.role
                ? ` · ${de.timeline.asRole(entry.role)}`
                : ""}
          </p>
          <div className="flex flex-wrap items-center gap-1">
            <a
              href={entry.source_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-h-11 items-center rounded-md px-3 underline underline-offset-4 hover:bg-line/40"
            >
              {de.timeline.pdf}{" "}
              <span className="sr-only">{de.timeline.newWindow}</span>
            </a>
            <ReportDialog
              paragraphId={entry.paragraph_id}
              topic={topic}
              context={date}
            />
          </div>
        </footer>
      </article>
    </li>
  );
}
