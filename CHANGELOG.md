# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `kurswechsel ingest`: downloads the Bundestag's plenary protocols (from 2017) and MdB master data and imports speeches paragraph by paragraph into SQLite with full-text search; interjections and the presiding officer's remarks are left out, quotations are flagged (#3)
- Repository bootstrap: Go backend and React frontend skeletons, `run-tests.sh` as the single test gate, CI, and Claude Code hooks that keep all work on issue branches (#1)
- Technology spikes with ADRs: SQLite with FTS5, the Bundestag protocol XML, a keyword prefilter, and qwen3:30b-a3b via Ollama as the stance classifier; hand-labelled Wehrpflicht gold set (#2)
