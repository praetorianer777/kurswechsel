# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `-prompt probs` and `-min-confidence`: one-letter answers with probabilities from token log-probabilities; `eval -dump` writes every answer for analysis (ADR 0007, #25)
- `-prompt label` for `classify` and `eval`: the model returns only relevance and stance, the quote is the first sentence with a topic keyword; 2–3× faster but flips more sides, so not the default (ADR 0006, #22)
- Architecture and methodology documentation; `kurswechsel eval -db … -gold-reviewed` stores results, and the „Methodik“ page shows measured accuracy only from a run against a reviewed gold set (#7)
- German website: search by name and topic, profiles, and timelines with original quotes, stance labels (text and icon, never colour alone), marked changes of position, PDF links and a "Fehler melden" dialog; mobile-first, WCAG 2.2 AA checked with axe in light and dark mode (#6)
- `kurswechsel seed-demo` and `scripts/build.sh`: fictional demo data and a single binary with the website embedded (#6)
- JSON API with topics, politician search (umlaut-tolerant), profiles and per-topic timelines that mark changes of position per session day; error reports with validation and a per-address rate limit; the built website is embedded into the binary (#5)
- `kurswechsel classify`: preselects paragraphs by topic keywords and classifies each speaker's stance (dafür, dagegen, neutral, unklar) with a verbatim quote and a German rationale; local Ollama models by default, the Claude API as an opt-in (#4)
- `kurswechsel eval`: scores a classifier against the hand-labelled gold set (#4)
- `kurswechsel ingest`: downloads the Bundestag's plenary protocols (from 2017) and MdB master data and imports speeches paragraph by paragraph into SQLite with full-text search; interjections and the presiding officer's remarks are left out, quotations are flagged (#3)
- Repository bootstrap: Go backend and React frontend skeletons, `run-tests.sh` as the single test gate, CI, and Claude Code hooks that keep all work on issue branches (#1)
- Technology spikes with ADRs: SQLite with FTS5, the Bundestag protocol XML, a keyword prefilter, and qwen3:30b-a3b via Ollama as the stance classifier; hand-labelled Wehrpflicht gold set (#2)
