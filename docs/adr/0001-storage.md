# 0001 — SQLite as the database

**Status:** Accepted · **Spike:** `spikes/storage`, `spikes/cmd/storagebench` · **Issue:** #2

## Context

Kurswechsel stores every speech since 2017 split into paragraphs, and needs
three things from the database: loading a whole legislative period in one go,
finding paragraphs by keyword, and returning one person's matching paragraphs
in date order for the timeline. The site is read-mostly; writes come from the
batch pipeline and the occasional error report.

## Measurements

All plenary protocols of WP19–21 (548 sessions, 65,380 speeches, 746,981
paragraphs), terms `wehrpflicht`, `wehrdienst`; query times are the median of
seven runs, on the development machine (12 cores, 31 GB RAM, NVMe).

| Engine | Load | Search | Hits | Timeline | Hits | Size |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| SQLite (modernc, FTS5) | 20.1 s | 0.54 ms | 678 | 2.5 ms | 40 | 406 MB |
| DuckDB (cgo, ILIKE scan) | 2.2 s | 651 ms | 700 | 37.5 ms | 40 | 130 MB |
| PostgreSQL 18 (tsvector german, GIN) | 67.5 s | 0.54 ms | 678 | 1.4 ms | 40 | 664 MB |

DuckDB matches substrings, so it also finds compounds such as
„Bundeswehrdienstzeit“; FTS5 and PostgreSQL match word prefixes.

## Decision

Use **SQLite** through the pure-Go driver `modernc.org/sqlite`, with an FTS5
index over paragraph text, WAL mode, and schema migrations embedded in the
binary.

## Reasons

- Search and timeline queries answer in milliseconds, as fast as PostgreSQL
  and two orders of magnitude faster than DuckDB's scan.
- No cgo and no server: the whole site is one binary plus one file, which
  keeps deployment, backups (`VACUUM INTO`) and CI trivial.
- Loading 750k paragraphs in 20 s is irrelevant for a nightly batch job, and
  new sessions arrive a few at a time.

## Consequences

- One writer at a time. Fine for a batch pipeline plus rare error reports;
  revisit if user-generated writes grow.
- DuckDB stays interesting for ad-hoc analysis (it reads SQLite files
  directly), but not as the serving database.
- If the project ever needs concurrent writers or several app servers,
  PostgreSQL is the measured fallback; the SQL is kept portable where it is
  cheap to do so.
