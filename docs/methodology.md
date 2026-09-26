# Methodology

Kurswechsel is only as credible as it is neutral. It shows evidence; the
reader judges. This document is the reference for the website's
[„Methodik“](../frontend/src/i18n/de.ts) page.

## Principles

1. **Nothing without evidence.** Every entry shows the original paragraph and
   links to the official protocol. A model's quote is shown only if it occurs
   verbatim in the paragraph.
2. **Neutral language.** „Positionswechsel“, never „Lüge“ or „Umfallen“.
   Stance colours carry no good/bad connotation (no red/green) and every
   stance also has its own icon and text.
3. **Same rules for everyone.** The classifier never sees the speaker's name
   or party. All factions go through the same pipeline and thresholds.
4. **Measure before publishing.** Accuracy is measured on a hand-labelled
   gold set and shown on the website only once a person has reviewed that set.
5. **Corrections.** Every entry has a „Fehler melden“ button; reports are
   stored with the paragraph they concern.

## What counts as a statement

- Only paragraphs the speaker says themselves. Interjections
  (`<kommentar>`), the presiding officer's remarks and footnotes are dropped
  by the parser.
- Quotations the speaker reads out (paragraph class `Z`) are never candidates.
  Paraphrased opposing views inside normal paragraphs remain a known source
  of error, which the `unklar` label and per-day comparison mitigate.

## Stance labels (topic „Wehrpflicht“)

| Label | Meaning |
| --- | --- |
| `dafuer` | The speaker supports compulsion: reinstating conscription, compulsory screening, a mandatory year of service, or criticises its suspension. |
| `dagegen` | The speaker opposes compulsion or insists on voluntary service only. |
| `neutral` | Relevant, but facts, procedure or a model without taking a side. |
| `unklar` | Irony, rhetorical questions or other people's views; no reliable reading. |

The full prompt is `stance.SystemPrompt` in
`backend/internal/stance/stance.go`, the topic definition is in
`backend/internal/topic/topic.go`. Changing either means bumping
`stance.PromptVersion`.

## Change of position

Compared per **session day**, not per paragraph. A day's position is the
majority of its `dafuer` and `dagegen` entries; ties and days with only
`neutral` or `unklar` entries have none and are skipped. A change is marked on
the first entry of a day whose position differs from the last day that had
one. Legitimate reasons for change — new facts, coalition agreements, crises —
are common; the marker states a difference, not a judgement.

## Measuring accuracy

- Gold set: [`evaluation/gold-wehrpflicht.jsonl`](../evaluation/README.md),
  110 real paragraphs from three sampling strata, pre-labelled and awaiting
  human review (#8).
- `kurswechsel eval` sends the gold items through the same keyword prefilter
  and classifier as production data and reports:
  - relevance precision and recall (a relevant paragraph the prefilter drops
    counts as missed),
  - stance accuracy and macro-F1 over the four labels,
  - **flip rate**: the share of `dafuer`/`dagegen` items labelled as the
    opposite — the error that would draw a false change of position,
  - the share of verbatim quotes.
- `kurswechsel eval -db … -gold-reviewed` stores a result for the website;
  without `-gold-reviewed` it is stored but never shown.

The model choice and the spike's comparison of models are in
[ADR 0004](adr/0004-stance-classifier.md).

### Latest measurement

Production code, `ollama/qwen3:30b-a3b`, prompt `v1`, 25 September 2026, on
the **not yet reviewed** gold set (46 paragraphs through the prefilter):

| Metric | Value |
| --- | ---: |
| Relevance precision | 0.97 |
| Relevance recall | 0.86 |
| Stance accuracy | 0.65 |
| Stance macro-F1 | 0.58 |
| Dafür/dagegen flipped | 2 of 17 (12 %) |
| Verbatim quotes | 31 of 33 |

The spike measured the same model with an earlier prompt wording at 0.61
accuracy and 1 of 16 flipped; with samples this small, differences of a few
paragraphs are noise. These figures are not shown on the website until #8 is
done.

## Known limitations

- Keyword preselection misses paragraphs that discuss a topic without its
  vocabulary; the gold set's third stratum exists to find them.
- Single paragraphs lack the context of the whole speech.
- The gold set is small (37 relevant paragraphs); accuracy figures have wide
  error margins until it grows.

## Legal

Plenary protocols and master data are official works and in the public domain
under § 5 UrhG. Only protocol text is used; press releases and interviews,
which are not, are out of scope. Imprint (§ 5 DDG) and privacy policy must be
completed before the site goes public.
