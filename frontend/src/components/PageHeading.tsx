import { useEffect, useRef, type ReactNode } from "react";
import { de } from "../i18n/de";

/**
 * The page's h1. It also sets the document title and, after client-side
 * navigation, takes focus so screen readers announce the new page instead of
 * staying silent on the link that was activated.
 */
export function PageHeading({
  title,
  children,
}: {
  title: string;
  children?: ReactNode;
}) {
  const ref = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    document.title = `${title} – ${de.appName}`;
    if (window.history.state?.idx > 0) ref.current?.focus();
  }, [title]);
  return (
    <h1
      ref={ref}
      tabIndex={-1}
      className="text-2xl font-bold tracking-tight text-balance focus:outline-none sm:text-3xl"
    >
      {children ?? title}
    </h1>
  );
}
