import type { ReactNode } from "react";
import { Link, NavLink } from "react-router";
import { de } from "../i18n/de";

const navLink = ({ isActive }: { isActive: boolean }) =>
  `inline-flex min-h-11 items-center rounded-md px-3 underline-offset-4 hover:underline ${
    isActive ? "font-semibold underline" : ""
  }`;

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <a
        href="#inhalt"
        className="sr-only z-50 rounded-md bg-surface px-4 py-3 font-semibold text-ink shadow-lg focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
      >
        {de.skipLink}
      </a>
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex max-w-4xl flex-wrap items-center justify-between gap-2 px-4 py-3">
          <Link
            to="/"
            className="inline-flex min-h-11 items-center text-xl font-bold tracking-tight"
          >
            {de.appName}
          </Link>
          <nav aria-label={de.nav.label}>
            <ul className="flex gap-1">
              <li>
                <NavLink to="/" end className={navLink}>
                  {de.nav.home}
                </NavLink>
              </li>
              <li>
                <NavLink to="/methodik" className={navLink}>
                  {de.nav.methodology}
                </NavLink>
              </li>
            </ul>
          </nav>
        </div>
      </header>
      <main
        id="inhalt"
        tabIndex={-1}
        className="mx-auto w-full max-w-4xl flex-1 px-4 py-8 focus:outline-none"
      >
        {children}
      </main>
      <footer
        aria-label={de.footer.label}
        className="border-t border-line bg-surface text-sm text-muted"
      >
        <div className="mx-auto flex max-w-4xl flex-col gap-3 px-4 py-6">
          <p>{de.footer.source}</p>
          <ul className="flex flex-wrap gap-x-2">
            <li>
              <Link
                to="/impressum"
                className="inline-flex min-h-11 items-center px-1 underline"
              >
                {de.footer.imprint}
              </Link>
            </li>
            <li>
              <Link
                to="/datenschutz"
                className="inline-flex min-h-11 items-center px-1 underline"
              >
                {de.footer.privacy}
              </Link>
            </li>
            <li>
              <a
                href="https://github.com/praetorianer777/kurswechsel"
                className="inline-flex min-h-11 items-center px-1 underline"
              >
                {de.footer.code}
              </a>
            </li>
          </ul>
        </div>
      </footer>
    </div>
  );
}
