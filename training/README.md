# Training data: Wehrpflicht

`wehrpflicht.jsonl` holds 400 real paragraphs from Bundestag plenary
protocols (WP19–21), drawn by `kurswechsel sample` from the keyword candidates
of the topic, spread over legislative periods and factions and **excluding
every paragraph of the gold set** in `../evaluation/`, so that training never
sees the test data.

## Status

Pre-labelled by Claude (`labeled_by: "claude"`) following the guide in
[`../evaluation/README.md`](../evaluation/README.md), awaiting human review:

| | Sure | Unsure |
| --- | ---: | ---: |
| dafür | 43 | 56 |
| dagegen | 72 | 19 |
| neutral | 11 | 35 |
| unklar | 1 | 34 |
| not relevant | 106 | 23 |

## Review

```bash
cd backend
go run ./cmd/kurswechsel review        # opens http://127.0.0.1:8090
```

The page shows every **unsure** pre-label and a random **sample of 50 sure
ones**, one at a time, with the pre-label selected; confirm or correct it.
Each click is saved to the file immediately. The header shows the agreement
rate on the sample. At ≥ 90 % the remaining sure pre-labels are accepted;
below that, the guide gets sharpened and the batch revisited (issue #27).

## Fields

`relevant`, `stance`, `note`, `unsure`, `labeled_by` are the pre-label;
`sampled` marks the agreement sample; `review` holds the reviewer's verdict,
which takes precedence (`training.Item.Final`).
