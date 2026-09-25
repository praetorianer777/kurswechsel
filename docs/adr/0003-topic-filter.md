# 0003 — Keyword prefilter, relevance decided by the classifier

**Status:** Accepted · **Spike:** `spikes/topicfilter`, `spikes/cmd/topicbench` · **Issue:** #2

## Context

Out of 750k paragraphs only a few hundred are about any one topic. The
pipeline needs a cheap first step that keeps almost every relevant paragraph,
so the expensive classifier only sees a small candidate set.

## Measurements

Gold set: 110 paragraphs, 37 relevant (see `spikes/testdata/README.md`).
Embeddings: `bge-m3` via local Ollama, cosine similarity to a topic question;
thresholds tuned on the gold set itself, so embedding rows are upper bounds.

| Strategy | Threshold | Precision | Recall | F1 | FP | FN |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| keywords | – | 0.80 | 0.97 | 0.88 | 9 | 1 |
| embedding | 0.508 | 0.83 | 0.81 | 0.82 | 6 | 7 |
| keywords AND embedding | 0.470 | 0.89 | 0.89 | 0.89 | 4 | 4 |
| keywords OR embedding | 0.541 | 0.80 | 1.00 | 0.89 | 9 | 0 |

Embedding all 111 texts took 7 s on the local GPU.

## Decision

Use a **per-topic keyword regex** as the prefilter, and let the **stance
classifier decide relevance** as part of its answer. Embeddings are not used.

## Reasons

- Keywords reach 0.97 recall at zero cost; the one miss (a paragraph that
  talks about being „zum Dienst an der Waffe verpflichtet“) is better fixed
  with a keyword than with a model.
- Embeddings lose recall on their own, and their precision gain duplicates
  what the classifier's relevance judgement does anyway.
- Fewer moving parts: no vector index, no embedding model to version.

## Consequences

- Recall is overestimated: the gold set was sampled partly with the same
  vocabulary. The duty stratum (lines 91–110) exists to find misses; keyword
  lists are reviewed whenever a new topic is added.
- Topic definitions carry their keyword list and are stored in the database,
  so every stored assignment can be traced to the list that produced it.
