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

cd backend && go run ./cmd/kurswechsel serve   # API on 127.0.0.1:8080
cd frontend && npm ci && npm run dev           # website with /api proxied to the backend
```

## Contributing

Every change starts as an English GitHub issue and lands through a pull request
from a branch named `<type>/<issue>-<slug>`. See `.claude/skills/gh/SKILL.md`.

## Licence

[GPL-3.0](LICENSE)
