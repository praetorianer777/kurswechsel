# Technology spikes

Throwaway code that measured the options behind the
[architecture decision records](../docs/adr). It is a separate Go module so
its database drivers never reach the production binary. `run-tests.sh` runs
its smoke tests so the numbers stay reproducible.

Every command reads or writes `../data/` (git-ignored).

| Command | ADR | What it does |
| --- | --- | --- |
| `go run ./cmd/fetch` | 0002 | Downloads all plenary protocols of WP19–21 |
| `go run -tags duckdb ./cmd/storagebench -pg postgres://…` | 0001 | Loads the corpus into SQLite, DuckDB and PostgreSQL and times the queries |
| `go run ./cmd/goldcandidates` | – | Draws the reproducible sample behind `../evaluation/gold-wehrpflicht.jsonl` |
| `go run ./cmd/topicbench` | 0003 | Scores keyword and embedding topic filters on the gold set |
| `go run ./cmd/stancebench -models a,b@think` | 0004 | Scores local Ollama models as stance classifiers |
| `python nli/nli_bench.py` | 0005 | Scores zero-shot NLI classifiers (Python; `pip install -r nli/requirements.txt`) |

The benchmarks call only a local Ollama server (`-ollama`, default
`http://127.0.0.1:11434`), never a paid API.

PostgreSQL for the storage benchmark:

```bash
docker run -d --rm --name kw-spike-pg -e POSTGRES_PASSWORD=spike -p 127.0.0.1:55432:5432 postgres:18-alpine
SPIKE_PG_URL=postgres://postgres:spike@127.0.0.1:55432/postgres go test -tags duckdb ./storage/
```
