# 0007 — Stance from token probabilities: fast, but no better and not calibrated

**Status:** Accepted, figures preliminary until the gold set is reviewed (#8) ·
**Issue:** #25 · **Builds on:** [0006](0006-label-only-prompt.md)

## Context

Decision models such as Jev (#24, dropped because it is cloud-only) return a
probability per answer option instead of text. The local counterpart: the
model answers a multiple-choice question with one letter (A = not relevant,
B = dafür, C = dagegen, D = neutral, E = unklar), and Ollama's token
log-probabilities give a probability per option. A threshold could then turn
uncertain sides into „unklar“, to avoid false changes of position.

## Measurements

`kurswechsel eval -prompt probs -dump …`, same 46 gold paragraphs and metrics
as ADR 0005/0006; thresholds swept on the dumped answers. „Right side“ counts
gold dafür/dagegen paragraphs labelled correctly (of 17).

| Model | Accuracy | Flipped | Right side | Relevance P / R | Per paragraph | Effect of a threshold 0.6–0.9 |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| qwen3:4b-instruct-2507 | 0.46 | 2 | 10 | 0.89 / 0.92 | 0.5 s | none until 0.9 (1 flip fewer) |
| qwen3:8b | 0.43 | 1 | 8 | 0.92 / 0.59 | 0.7 s | removes the flip |
| mistral-nemo:12b | 0.46 | 1 | 8 | 0.83 / 0.92 | 0.9 s | removes the flip and half of the right sides |
| gemma3:12b | 0.38 | 3 | 11 | 0.90 / 0.97 | 1.4 s | none |
| qwen3:14b | 0.59 | 2 | 12 | 0.81 / 0.95 | 0.9 s | at 0.9: 1 flip fewer, 1 right side fewer |
| mistral-small3.2:24b | 0.62 | 0 | 10 | 0.85 / 0.59 | 1.9 s | only removes right sides |
| qwen3:30b-a3b | – | – | – | – | – | not usable, see below |
| *Default: qwen3:30b-a3b, prompt v1 (ADR 0005)* | *0.65* | *2* | *12* | *0.97 / 0.86* | *4.2 s* | |

qwen3:30b-a3b ignores `think: false` when no JSON schema is set and starts
reasoning in plain text; with a schema of the five letters Ollama 0.30
reports the log-probabilities *before* the schema is applied, so the forced
letter has no usable probability (it answered „B = no“ to „Is Berlin the
capital of Germany?“).

## Decision

Keep prompt v1 with qwen3:30b-a3b as the default. Keep `-prompt probs` as an
option; no confidence threshold by default.

## Reasons

- No model beats the default's accuracy; the best, mistral-small3.2, comes
  close (0.62) and flipped no side, but misses 41 % of the relevant
  paragraphs, which would thin out the timelines.
- The probabilities are not calibrated: most answers come with 90–100 %, so
  a threshold rarely changes anything, and where it does it removes correct
  sides as often as wrong ones.
- It is 2–8× faster than prompt v1, which makes it useful for quick passes.

## Consequences

- Across ADRs 0004–0007 the zero-shot options on this hardware are
  exhausted: prompt variants and models change accuracy by a few paragraphs
  of 46 and trade flips against recall.
- The next step with the most potential is training on labelled data
  (public x-stance plus corrected in-domain examples), not another prompt.
