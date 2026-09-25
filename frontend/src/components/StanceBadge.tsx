import type { Stance } from "../api/types";
import { de } from "../i18n/de";

const styles: Record<Stance, string> = {
  dafuer: "bg-pro-soft text-pro border-pro",
  dagegen: "bg-contra-soft text-contra border-contra",
  neutral: "bg-neutral-soft text-neutral border-neutral",
  unklar: "bg-unclear-soft text-unclear border-unclear",
};

/** Stance shapes differ as well as colours, so the badge never relies on colour alone. */
function Icon({ stance }: { stance: Stance }) {
  const common = {
    width: 16,
    height: 16,
    viewBox: "0 0 16 16",
    "aria-hidden": true,
    focusable: false,
    className: "shrink-0",
  } as const;
  switch (stance) {
    case "dafuer":
      return (
        <svg {...common}>
          <circle
            cx="8"
            cy="8"
            r="7"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          />
          <path d="M8 4.5v7M4.5 8h7" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "dagegen":
      return (
        <svg {...common}>
          <rect
            x="1"
            y="1"
            width="14"
            height="14"
            rx="2"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          />
          <path d="M4.5 8h7" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "neutral":
      return (
        <svg {...common}>
          <path
            d="M8 1.5 14.5 8 8 14.5 1.5 8Z"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          />
        </svg>
      );
    case "unklar":
      return (
        <svg {...common}>
          <circle
            cx="8"
            cy="8"
            r="7"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeDasharray="2.5 2"
          />
          <circle cx="8" cy="8" r="1.4" fill="currentColor" />
        </svg>
      );
  }
}

export function StanceBadge({ stance }: { stance: Stance }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-sm font-semibold ${styles[stance]}`}
    >
      <Icon stance={stance} />
      {de.stance[stance]}
    </span>
  );
}

export function StanceLegend() {
  const stances: Stance[] = ["dafuer", "dagegen", "neutral", "unklar"];
  return (
    <section
      aria-labelledby="legende"
      className="rounded-lg border border-line p-4"
    >
      <h2 id="legende" className="mb-2 font-semibold">
        {de.legend}
      </h2>
      <dl className="grid gap-2 sm:grid-cols-2">
        {stances.map((s) => (
          <div key={s} className="flex flex-wrap items-center gap-2">
            <dt>
              <StanceBadge stance={s} />
            </dt>
            <dd className="text-sm text-muted">{de.stanceHelp[s]}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
