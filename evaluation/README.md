# Gold set: Wehrpflicht

`gold-wehrpflicht.jsonl` holds 110 real paragraphs from Bundestag plenary
protocols (WP19–21), drawn reproducibly by `go run ./cmd/goldcandidates` in
`spikes/` from three strata:

| Lines | Stratum | Purpose |
| --- | --- | --- |
| 1–45 | Explicit conscription vocabulary (`wehrpflicht`, `wehrdienst`, `musterung`, …) | Stance labels, keyword precision |
| 46–90 | Wider defence debate without that vocabulary | Hard negatives |
| 91–110 | Talk of duties and service without that vocabulary | Relevant paragraphs the keywords miss |

The protocols are official documents and are in the public domain under § 5 UrhG.

## Labelling guide

**Topic:** whether military service, or a general duty to serve that includes it, should be compulsory in Germany.

- `relevant` — the paragraph substantively discusses that question, including the 2025 military service law. A passing mention (history, other countries, compensation for service-related injuries) is not relevant.
- `stance` (relevant paragraphs only), judged from the paragraph alone:
  - `dafuer` — supports compulsion: reinstating conscription, compulsory screening, a mandatory questionnaire, a mandatory year of service, compulsion as a fallback („zunächst freiwillig, notfalls verpflichtend“, keeping the Wehrpflicht in the Grundgesetz), or criticises its suspension. Supporting the 2025 Wehrdienst law, which includes compulsory registration and screening, counts too.
  - `dagegen` — opposes compulsion or insists on voluntary service only.
  - `neutral` — relevant, but states facts, procedure or a model without taking a side on compulsion.
  - `unklar` — a position may be implied but cannot be read off reliably: irony, rhetorical questions, other people's views.
- `note` explains non-obvious decisions.

**Status:** pre-labelled during the technology spike and awaiting human review
(issue #8). Do not publish accuracy figures derived from it before that review.
