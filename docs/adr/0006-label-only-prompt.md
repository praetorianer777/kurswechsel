# 0006 — Label-only prompt with rule-based quotes: kept as an option, not the default

**Status:** Accepted, figures preliminary until the gold set is reviewed (#8) ·
**Issue:** #22 · **Builds on:** [0005](0005-specialised-and-smaller-models.md)

## Context

Prompt v1 asks the model for relevance, stance, a verbatim quote and a German
rationale at once. The hypothesis: with only `{relevant, stance}` to produce,
smaller models would classify better, and the quote can be chosen by rule —
the first sentence containing a topic keyword, verbatim by construction.

## Measurements

Same 46 gold paragraphs and metrics as ADR 0005; `kurswechsel eval -prompt
label` against the full prompt v1. „Flipped“ is out of 17 gold dafür/dagegen
paragraphs.

| Model | v1 accuracy | v1 flipped | v1 per paragraph | label accuracy | label flipped | label per paragraph |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| qwen3:4b-instruct-2507 | 0.41 | 2 | 2.3 s | 0.49 | 4 | 0.7 s |
| qwen3:8b | 0.41 | 2 | 2.3 s | 0.38 | 3 | 0.9 s |
| mistral-nemo:12b | 0.57 | 2 | 2.4 s | 0.62 | 3 | 1.0 s |
| gemma3:12b | 0.46 | 3 | 3.5 s | 0.43 | 5 | 1.7 s |
| qwen3:14b | 0.54 | 1 | 3.7 s | 0.54 | 3 | 1.2 s |
| mistral-small3.2:24b | 0.54 | 1 | 7.2 s | 0.54 | 4 | 3.1 s |
| **qwen3:30b-a3b** | **0.65** | **2** | 4.2 s | 0.38 | 4 | 2.8 s |

## Decision

Keep prompt v1 and qwen3:30b-a3b as the default. Keep the label-only prompt
as an option (`-prompt label`, stored as `ollama/<model>+label`, prompt
version `v2-label`).

## Reasons

- Every model flips more sides with the label-only prompt (3–5 instead of
  1–3 of 17). One paragraph more would be noise; all seven moving the same way
  is not. Flipped sides are the error that draws false changes of position.
- The default model loses most (0.65 → 0.38). The instructions that come with
  the quote and the rationale — the shortest passage carrying the decision,
  the speaker's own position — evidently help the model read the task, even
  though the schema asks for the label first.
- The gain is speed: 2–3× faster. mistral-nemo:12b reaches 0.62 at 1 s per
  paragraph and fits into 7 GB, which makes it useful for quick first passes
  over new topics.

## Consequences

- The website keeps showing a rationale for every entry.
- For #11, the more promising direction is the opposite: ask for the quote
  *before* the label so the label rests on the quoted words.
