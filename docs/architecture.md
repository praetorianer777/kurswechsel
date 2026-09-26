# Architecture

Kurswechsel is one Go binary with the website embedded, one SQLite file, and
an optional local LLM server. Everything runs on a single machine.

```
Bundestag open data ──► ingest ──► SQLite ◄── classify ◄──► Ollama (local LLM)
 (protocol XML,            │        (FTS5)        │
  MdB master data)         │          ▲           └── eval ◄── gold set
                           │          │
                           │     serve (JSON API + embedded React site) ◄── browser
```

## Pipeline

| Step | Command | Package | Notes |
| --- | --- | --- | --- |
| 1. Collect | `kurswechsel ingest` | `internal/bundestag`, `internal/ingest` | Downloads each session's XML once into a cache, then imports only sessions the database lacks. Streaming parser; interjections, footnotes and the presiding officer are dropped, quotations (`klasse="Z"`) flagged. |
| 2. Assign topics | `kurswechsel classify` | `internal/topic`, `internal/store` | Keyword regex per topic over every non-quotation paragraph ([ADR 0003](adr/0003-topic-filter.md)). |
| 3. Classify | `kurswechsel classify` | `internal/stance`, `internal/classify` | One LLM call per candidate with a versioned prompt and a JSON schema; stores stance, verbatim-checked quote, German rationale, model and prompt version. Resumable; stops after five failures in a row. |
| 4. Measure | `kurswechsel eval` | `internal/eval` | Runs a classifier over the gold set through the same prefilter; accuracy, macro-F1, flip rate, confusion matrix. |
| 5. Serve | `kurswechsel serve` | `internal/api`, `internal/timeline`, `internal/web` | JSON API, change-of-position detection per session day, error reports, the embedded website. |

## Data model

```
politicians ──< memberships          (faction history from the master data)
     │
     └──< speeches >── sessions      (one row per <rede>; faction at the time)
              │
              └──< paragraphs ──< paragraph_topics >── topics
                       │                                   │
                       └──< stances >──────────────────────┘
                       └──< reports
evaluations                           (eval results; only reviewed runs are published)
```

- `paragraphs(speech_id, position)` is unique and upserted, so a re-import
  keeps paragraph IDs and everything that refers to them.
- `paragraphs_fts` is an FTS5 index kept in sync by triggers.
- `stances` holds one classification per paragraph and topic; a new
  `stance.PromptVersion` makes old rows pending again.
- Migrations are embedded SQL files applied in order at start-up
  (`internal/store/migrations`).

## Website

React 19, TypeScript, React Router and Tailwind CSS 4, built by Vite and
embedded into the binary (`scripts/build.sh`). All visitor-facing text lives in
`frontend/src/i18n/de.ts`; the API returns error codes, the site translates
them. There is no client-side state beyond the URL: search terms and topics
are query parameters, so every view can be linked.

## Decisions

See the [ADRs](adr/README.md) for storage, data source, topic filter and
classifier choice, each with the measurements behind it.

## Operations

- **Deploy:** copy `backend/bin/kurswechsel` and the database; run
  `kurswechsel serve -addr :8080 -db kurswechsel.db` behind a TLS proxy.
- **Update:** `ingest` then `classify` (e.g. nightly); both only touch what
  is new.
- **Backups:** `sqlite3 kurswechsel.db "VACUUM INTO 'backup.db'"` while serving.
- **Rate limiting** of error reports is per client address in memory; behind a
  proxy every request shares the proxy's address, so the limit then applies
  to all visitors together until the proxy's address header is trusted.
