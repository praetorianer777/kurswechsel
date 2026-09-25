# Kurswechsel

Kurswechsel shows what members of the German Bundestag said about a topic over
time: every statement in chronological order, with the original quote and a
link to the plenary protocol, and changes of position made visible — without
passing judgement. The website is in German; code and documentation are in
English.

> Status: early development. See the [issues](https://github.com/praetorianer777/kurswechsel/issues) for the roadmap.

## How it works

1. **Collect** speeches from the Bundestag's open data (plenary protocols since 2017).
2. **Assign topics** to paragraphs, starting with *Wehrpflicht* (conscription).
3. **Classify the stance** of each relevant paragraph (for, against, neutral, unclear) with an LLM, keeping the quoted passage and a rationale.
4. **Show a timeline** per person and topic and mark changes of position.

## Repository layout

| Path | Contents |
| --- | --- |
| `backend/` | Go module: ingestion, classification, JSON API, single `kurswechsel` binary |
| `frontend/` | React + TypeScript + Tailwind CSS website (German UI) |
| `docs/` | Architecture, methodology and architecture decision records |
| `run-tests.sh` | The one test gate, used by the git hook and CI |

## Development

Requirements: Go ≥ 1.26, Node.js ≥ 24, `jq`, and `gh` for the workflow.

```bash
./run-tests.sh                          # everything CI runs

cd backend
go run ./cmd/kurswechsel ingest -db ../data/kurswechsel.db -cache ../data/raw   # ≈ 5 min download, 2 min import
go run ./cmd/kurswechsel classify -db ../data/kurswechsel.db                     # local Ollama model
go run ./cmd/kurswechsel eval                                                   # accuracy on the gold set
go run ./cmd/kurswechsel serve                                                  # API on 127.0.0.1:8080

cd frontend && npm ci && npm run dev    # website with /api proxied to the backend
```

`ingest` downloads the plenary protocols of the 19th–21st legislative
periods (≈ 470 MB) and the MdB master data into the cache once, then imports
only sessions the database does not have yet. `-offline` works from the cache
alone; `-force` re-imports everything without changing paragraph IDs.

`classify` and `eval` run against a local [Ollama](https://ollama.com) server
by default and never call a paid API unless given `-provider claude` (which
reads `ANTHROPIC_API_KEY`). Only paragraphs with a new or outdated
classification are sent to the model, so interrupted runs resume.

## Contributing

Every change starts as an English GitHub issue and lands through a pull request
from a branch named `<type>/<issue>-<slug>`. See `.claude/skills/gh/SKILL.md`.

## Licence

[GPL-3.0](LICENSE)
