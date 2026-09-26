# 0008 — Surrounding paragraphs as context: promising, decision after the gold review

**Status:** Proposed — decide after #8 · **Issue:** #11 · **Builds on:** [0005](0005-specialised-and-smaller-models.md)

## Context

Reviewing training data (#27) showed that many paragraphs cannot be judged
alone: they answer the previous speaker, continue the paragraph before, or
refer to „das“ without saying what it is. The classifier has the same
handicap. Prompt v1 with context gives the model the two paragraphs before
and one after from the same speech and, at the start of a speech, the end of
the previous one, marked as context only; it classifies and quotes only the
marked paragraph (`kurswechsel eval -context <db>`, prompt `v1-context`).

## Measurements

Same 46 gold paragraphs and metrics as ADR 0005; all 46 were found in the
database and got their context. „Flipped“ of 17 sided gold paragraphs.

| Model | v1 accuracy | v1 flipped | context accuracy | context flipped | context relevance P / R | context per paragraph |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| qwen3:4b-instruct-2507 | 0.41 | 2 | 0.49 | 2 | 0.86 / 1.00 | 2.5 s |
| qwen3:8b | 0.41 | 2 | 0.54 | 1 | 0.97 / 0.92 | 3.1 s |
| mistral-nemo:12b | 0.57 | 2 | 0.59 | 4 | 0.85 / 0.92 | 3.0 s |
| gemma3:12b | 0.46 | 3 | 0.43 | 2 | 0.86 / 1.00 | 4.8 s |
| **qwen3:14b** | 0.54 | 1 | **0.62** | **0** | 0.92 / 0.92 | 4.3 s |
| mistral-small3.2:24b | 0.54 | 1 | 0.54 | 2 | 0.92 / 0.92 | 8.0 s |
| qwen3:30b-a3b (default) | **0.65** | 2 | 0.51 | 1 | 0.97 / 1.00 | 5.4 s |

With context, sided paragraphs are recognised better (qwen3:14b: 9 instead of
7 of 11 „dafür“ right, no side flipped), but paragraphs labelled „neutral“
are more often given a side (qwen3:30b-a3b: 4 instead of 10 of 16 right).

## Why the numbers are biased against context

- The gold labels were made from the paragraph alone. When context shows
  that a factual paragraph belongs to a clear speech for or against, the
  model's side may be right and still count as an error.
- The labelling rule added in #27 (any supported compulsion, including a
  fallback or the 2025 Wehrdienst law, counts as „dafür“) is not yet applied
  to the gold set; six gold paragraphs labelled „neutral“ are likely „dafür“
  under it (listed on #8) — exactly the kind of case now counted as wrong.

## Decision

Keep qwen3:30b-a3b with prompt v1 for now. Treat **qwen3:14b with context**
as the leading candidate: it fits completely into 16 GB VRAM, flipped no side
and comes close in accuracy. Decide after the gold set has been reviewed with
context and the new rule (#8), then re-run this table.

## Consequences

- `ClassifyInContext` and `v1-context` exist for Ollama; `kurswechsel
  classify` does not use context yet. If context wins, the production
  pipeline gets it in the same change that switches the default.
- The review page (#27) should also serve the gold set, with context, for #8.
