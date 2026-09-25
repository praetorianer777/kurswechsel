# 0002 — Bundestag plenary protocol XML as the primary source

**Status:** Accepted · **Spike:** `spikes/corpus`, `spikes/cmd/fetch` · **Issue:** #2

## Context

Every statement on the timeline must link back to the official record. The
candidates were the Bundestag's own open data and the *Parliamentary
Discourse Dataset* (Njie, Torkayesh, Venghaus, RWTH Aachen; Zenodo record
21258818, v2 of 8 July 2026).

## Comparison

| | Bundestag XML (`dserver.bundestag.de/btp/{wp}/{wp}{nnn}.xml`) | Parliamentary Discourse Dataset |
| --- | --- | --- |
| Coverage | WP19 onwards in the structured format (since 2017) | WP1–21, 1949 to April 2026 |
| Speaker identity | `redner id` = MdB master-data ID | `stammdaten_id` |
| Speech ID | Stable `rede id` (e.g. `ID2000700100`) | Dataset-internal |
| Paragraphs | Yes (`<p klasse="J">`), interjections separate (`<kommentar>`), presiding officer marked (`<name>`) | Full text per speech, no paragraph structure |
| Deep link to source | Session XML and PDF, speech ID | Via session reference |
| Updates | Published by the Bundestag after each session | Occasional releases |
| Size | 548 files, 473 MB for WP19–21 | 1.1 GB Parquet |
| Licence | Official works, public domain under § 5 UrhG | CC BY 4.0 |
| Measured effort | Download 548 files in ≈ 4.5 min at 0.3 s intervals; parse all in 14 s with `encoding/xml` | Not needed |

## Decision

Use the **Bundestag's plenary protocol XML** from WP19 onwards, together with
the **MdB master data** (`MdB-Stammdaten.zip`) for names and party history.

## Reasons

- Paragraphs and interjections are marked up, which the classifier needs:
  interjections and the presiding officer's remarks must never be attributed
  to the speaker.
- Stable speech IDs and official URLs make every timeline entry verifiable.
- New sessions appear within days, so the site can stay current.
- The spike parser is 150 lines; the structured format removes all PDF parsing.

## Consequences

- Nothing before 2017 for now. The Parliamentary Discourse Dataset is the
  planned way to add older periods later, matched through `stammdaten_id`.
- Downloads are polite (sequential, identifying user agent, delay between
  requests) and cached; each file is fetched once.
