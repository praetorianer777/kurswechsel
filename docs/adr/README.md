# Architecture decision records

Each record states a decision, the measurements behind it, and what it costs.
The measurements come from the reproducible spikes in [`spikes/`](../../spikes).

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-storage.md) | SQLite (pure Go, FTS5) as the database | Accepted |
| [0002](0002-data-source.md) | Bundestag plenary protocol XML as the primary source | Accepted |
| [0003](0003-topic-filter.md) | Keyword prefilter, relevance decided by the classifier | Accepted |
| [0004](0004-stance-classifier.md) | Local Ollama model (qwen3:30b-a3b) as the default stance classifier | Accepted, figures preliminary until #8 |
