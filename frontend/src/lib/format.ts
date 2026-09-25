const long = new Intl.DateTimeFormat("de-DE", {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});

/** Formats an ISO date as "1. März 2019". */
export function formatDate(iso: string): string {
  return long.format(new Date(iso));
}

/** The date part of an ISO timestamp, for <time dateTime>. */
export function isoDay(iso: string): string {
  return iso.slice(0, 10);
}
