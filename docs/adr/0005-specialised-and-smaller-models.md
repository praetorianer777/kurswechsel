# 0005 — Specialised classifiers and models that fit in 16 GB VRAM

**Status:** Accepted, figures preliminary until the gold set is reviewed (#8) ·
**Spike:** `spikes/nli`, `kurswechsel eval` · **Issue:** #19 · **Builds on:** [0004](0004-stance-classifier.md)

## Context

ADR 0004 chose qwen3:30b-a3b, an 18 GB model that does not fit into the
development machine's 16 GB of VRAM (about a quarter runs on the CPU). Two
questions remained: do small classifiers built for this kind of task do
better, and does a model that fits entirely into VRAM come close?

## Measurements

All runs use the 46 gold paragraphs that pass the keyword prefilter, and the
metrics of `kurswechsel eval`. „Flipped“ counts gold dafür/dagegen paragraphs
labelled as the opposite side. Same machine as ADR 0004, Ollama 0.30.7.

### Zero-shot NLI classifiers

Each label becomes a German hypothesis (English for Political DEBATE) and the
model scores entailment on the CPU (`spikes/nli/nli_bench.py`). „Tuned“
thresholds are fitted on the gold set itself and therefore an upper bound.

| Model | Thresholds | Stance accuracy | Macro-F1 | Flipped | Relevance P / R | Per paragraph (CPU) |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| mDeBERTa-v3-base-xnli-multilingual | fixed | 0.32 | 0.32 | 7 of 17 | 0.80 / 0.89 | 3.0 s |
| | tuned | 0.43 | 0.30 | 7 of 17 | 0.87 / 0.89 | |
| bge-m3-zeroshot-v2.0 | fixed | 0.41 | 0.36 | 2 of 17 | 0.81 / 0.70 | 4.4 s |
| | tuned | 0.59 | 0.46 | 4 of 17 | 0.83 / 0.65 | |
| Political_DEBATE_large_v1.0 (English) | fixed = tuned | 0.46 | 0.36 | 1 of 17 | 0.80 / 0.97 | 1.8 s |

`svalabs/gbert-large-zeroshot-nli`, a German NLI model, is no longer
available on Hugging Face.

### Local LLMs through the production classifier

`kurswechsel eval -model <name>`, production prompt v1, thinking off.

| Model | Size | Stance accuracy | Macro-F1 | Flipped | Verbatim quotes | Relevance P / R | Per paragraph |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| **qwen3:30b-a3b** (default) | 18 GB¹ | **0.65** | **0.58** | 2 of 17 | 31 of 33 | 0.97 / 0.86 | 4.2 s |
| mistral-nemo:12b | 7 GB | 0.57 | 0.46 | 2 of 17 | 34 of 41 | 0.83 / 0.92 | 2.4 s |
| qwen3:14b | 9 GB | 0.54 | 0.46 | 1 of 17 | 33 of 33 | 0.94 / 0.84 | 3.7 s |
| mistral-small3.2:24b | 15 GB | 0.54 | 0.42 | 1 of 17 | 40 of 42 | 0.88 / 1.00 | 7.2 s |
| gemma3:12b | 8 GB | 0.46 | 0.37 | 3 of 17 | 35 of 41 | 0.90 / 1.00 | 3.5 s |
| qwen3:8b | 5 GB | 0.41 | 0.32 | 2 of 17 | 35 of 36 | 0.89 / 0.86 | 2.3 s |
| qwen3:4b-instruct-2507 | 2.5 GB | 0.41 | 0.33 | 2 of 17 | 32 of 33 | 0.94 / 0.84 | 2.3 s |

¹ Mixture of experts with 3 B active parameters, so the part on the CPU
costs little.

## Decision

Keep **qwen3:30b-a3b** as the default. Do not use NLI classifiers.

## Reasons

- No model that fits into VRAM and no NLI classifier reaches its stance
  accuracy; most of their errors confuse „neutral“ with a side.
- NLI classifiers give no quote and no rationale, which the website needs for
  every entry, and are weaker than the LLMs even with thresholds tuned on the
  test set.
- qwen3:14b and mistral-small3.2 flip one side fewer, but with 17 sided
  paragraphs that is a difference of one paragraph — within noise.

## Consequences

- qwen3:14b is the recommended fallback for machines where the default is
  too slow: it fits into 9 GB, quotes verbatim every time and flipped the
  fewest sides; `kurswechsel classify -model qwen3:14b` switches to it.
- All figures rest on the unreviewed gold set (#8); re-run both tables after
  the review, before choosing between models this close together.
- Prompt work (#11) is likely worth more than a model change at this point.
