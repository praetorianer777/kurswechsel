# 0004 — Local Ollama model as the default stance classifier

**Status:** Accepted, figures preliminary until the gold set is reviewed (#8) ·
**Spike:** `spikes/stance`, `spikes/cmd/stancebench` · **Issue:** #2

## Context

Each candidate paragraph needs a stance label (dafür, dagegen, neutral,
unklar), a verbatim quote and a short German rationale. The classifier is
pluggable; the project runs on local models to avoid cloud costs, with the
Claude API available as an opt-in adapter.

## Measurements

The 45 gold paragraphs that pass the keyword prefilter (36 relevant), one
fixed prompt with a JSON schema, temperature 0, run on the development machine
(AMD Radeon RX 9070, 16 GB VRAM; Ollama 0.30.7). A relevant paragraph the
model calls irrelevant counts as `neutral` for stance accuracy, since it
disappears from the timeline. „Flipped“ counts gold dafür/dagegen paragraphs
labelled as the opposite side — the error that would draw a false change of
position.

| Model | Stance accuracy | Macro-F1 | Flipped | Verbatim quotes | Relevance P / R | Median latency |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| llama3.1:8b | 0.50 | 0.40 | 5 of 16 | 69 % | 0.81 / 0.83 | 1.5 s |
| qwen2.5:14b-instruct | 0.53 | 0.42 | 1 of 16 | 58 % | 0.91 / 0.81 | 2.8 s |
| **qwen3:30b-a3b** | **0.61** | **0.51** | **1 of 16** | **96 %** | **1.00** / 0.69 | 3.9 s |
| qwen3.5:4b, thinking | 0.62¹ | – | 0 of 7¹ | – | 0.93 / 0.88¹ | 75 s |
| qwen3:30b-a3b, thinking | – | – | – | – | – | 69–137 s² |

¹ Stopped after 19 of 45 paragraphs; 6 of 24 replies were truncated JSON.
² Stopped after two paragraphs: at this speed the 700+ Wehrpflicht
candidates would take more than a day.

Not run:

- `phi4:14b` crashes the Ollama runner with a JSON schema.
- `qwen3.5:9b`, `qwen3.6:27b` and `gpt-oss:20b` are thinking models. Ollama
  0.30 drops the schema for them unless thinking stays on, and thinking cost
  60–140 s per paragraph in the two runs above.
- Claude (`claude-opus-5` and cheaper tiers) is wired up as an opt-in
  provider (`-provider claude`) but was not measured, to keep the project
  free of cloud costs.

Confusion matrix of qwen3:30b-a3b (rows: gold, columns: predicted):

| | dafuer | dagegen | neutral | unklar |
| --- | ---: | ---: | ---: | ---: |
| dafuer | 7 | 1 | 3 | 0 |
| dagegen | 0 | 5 | 0 | 0 |
| neutral | 2 | 2 | 10 | 2 |
| unklar | 2 | 0 | 2 | 0 |

## Decision

Use **qwen3:30b-a3b via Ollama, thinking off** as the default classifier
(`kurswechsel classify` without flags).

## Reasons

- Best stance accuracy of the practical models and, together with
  qwen2.5, the fewest flipped sides — the error that matters most for a site
  about changes of position.
- 96 % of its quotes occur verbatim, so nearly every entry can show its
  evidence; the website hides the rest anyway.
- Perfect relevance precision: what it puts on a timeline is about the topic.
- Under 4 s per paragraph, so a topic's ~750 candidates classify in about
  an hour on one consumer GPU.

## Consequences

- **Relevance recall is 0.69**: about a third of relevant paragraphs are
  called irrelevant and do not appear. That hides statements rather than
  misrepresenting them, but it is the first thing to improve (prompt work,
  tracked in its own issue).
- 61 % accuracy is not good enough to publish without the safeguards the
  site already has: the disclaimer on every timeline, the original paragraph
  next to every label, per-day comparison, and error reports.
- Every figure here rests on a gold set labelled in one pass by one
  annotator. The review in #8 may move them; `kurswechsel eval` reproduces
  the measurement with the production code.
- Re-run this comparison when Ollama fixes schemas for thinking models or
  new local models appear; switching models is a flag.
