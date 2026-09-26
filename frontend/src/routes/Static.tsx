import { Link } from "react-router";
import { PageHeading } from "../components/PageHeading";
import { de } from "../i18n/de";

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
