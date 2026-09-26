"""Zero-shot NLI classifiers as stance detectors (issue #19).

Each label becomes a hypothesis; the model scores whether the paragraph
entails it. Metrics follow `kurswechsel eval`: the same keyword prefilter,
a relevant paragraph judged irrelevant counts as "neutral", and "flipped"
counts gold dafuer/dagegen items predicted as the opposite side.
"""

import argparse
import itertools
import json
import re
import time
from collections import Counter
from pathlib import Path

from transformers import pipeline

KEYWORDS = re.compile(
    r"wehrpflicht|wehrdienst|musterung|dienstpflicht|pflichtdienst|gesellschaftsjahr|pflichtjahr|dienst an der waffe",
    re.IGNORECASE,
)
LABELS = ["dafuer", "dagegen", "neutral", "unklar"]

HYPOTHESES = {
    "de": {
        "rel": "In diesem Text geht es darum, ob Wehrdienst oder ein Dienst für die Gesellschaft verpflichtend sein soll.",
        "pro": "Die sprechende Person ist dafür, dass Wehrdienst oder ein allgemeiner Dienst verpflichtend ist.",
        "con": "Die sprechende Person ist dagegen, dass Wehrdienst oder ein allgemeiner Dienst verpflichtend ist.",
    },
    "en": {
        "rel": "This text is about whether military service or national service should be compulsory.",
        "pro": "The speaker supports compulsory military service or compulsory national service.",
        "con": "The speaker opposes compulsory military service or compulsory national service.",
    },
}

MODELS = [
    ("MoritzLaurer/mDeBERTa-v3-base-xnli-multilingual-nli-2mil7", "de"),
    ("MoritzLaurer/bge-m3-zeroshot-v2.0", "de"),
    ("mlburnham/Political_DEBATE_large_v1.0", "en"),
]


def decide(p, rel_t, side_t, margin):
    side = max(p["pro"], p["con"])
    relevant = p["rel"] >= rel_t or side >= side_t
    if side < side_t:
        return relevant, "neutral"
    if abs(p["pro"] - p["con"]) < margin:
        return relevant, "unklar"
    return relevant, "dafuer" if p["pro"] > p["con"] else "dagegen"


def score(items, probs, rel_t, side_t, margin):
    tp = fp = fn = 0
    conf = Counter()
    for it in items:
        p = probs.get(it["id"])
        relevant, stance = decide(p, rel_t, side_t, margin) if p else (False, "neutral")
        tp += it["relevant"] and relevant
        fp += (not it["relevant"]) and relevant
        fn += it["relevant"] and not relevant
        if it["relevant"]:
            conf[(it["stance"], stance if relevant else "neutral")] += 1
    total = sum(conf.values())
    acc = sum(conf[(l, l)] for l in LABELS) / total
    f1s = []
    for l in LABELS:
        t = conf[(l, l)]
        fpl = sum(conf[(o, l)] for o in LABELS if o != l)
        fnl = sum(conf[(l, o)] for o in LABELS if o != l)
        if t + fpl + fnl:
            f1s.append(2 * t / (2 * t + fpl + fnl))
    sided = sum(v for (g, _), v in conf.items() if g in ("dafuer", "dagegen"))
    flips = conf[("dafuer", "dagegen")] + conf[("dagegen", "dafuer")]
    return {
        "acc": acc,
        "f1": sum(f1s) / len(f1s),
        "flips": f"{flips} of {sided}",
        "rel_p": tp / max(tp + fp, 1),
        "rel_r": tp / max(tp + fn, 1),
        "conf": conf,
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--gold", default="../../evaluation/gold-wehrpflicht.jsonl")
    ap.add_argument("--models", nargs="*", default=[m for m, _ in MODELS])
    ap.add_argument("--dump", default="")
    args = ap.parse_args()

    items = [json.loads(l) for l in Path(args.gold).read_text().splitlines() if l.strip()]
    candidates = [it for it in items if KEYWORDS.search(it["text"])]
    lang = dict(MODELS)
    rows, dump = [], {}
    for name in args.models:
        hyp = HYPOTHESES[lang.get(name, "de")]
        try:
            clf = pipeline("zero-shot-classification", model=name, device="cpu")
        except OSError as err:
            print(f"{name}: skipped, cannot load: {err}", flush=True)
            continue
        start = time.perf_counter()
        probs = {}
        for it in candidates:
            out = clf(it["text"], candidate_labels=list(hyp.values()), hypothesis_template="{}", multi_label=True)
            by = dict(zip(out["labels"], out["scores"]))
            probs[it["id"]] = {k: by[v] for k, v in hyp.items()}
        per_item = (time.perf_counter() - start) / len(candidates)
        dump[name] = probs
        fixed = score(items, probs, 0.5, 0.5, 0.2)
        grid = itertools.product([0.3, 0.5, 0.7, 0.9], [0.3, 0.5, 0.7, 0.9], [0.0, 0.1, 0.2, 0.4])
        best = max((score(items, probs, *g) | {"params": g} for g in grid), key=lambda s: s["acc"])
        rows.append((name, fixed, best, per_item))
        print(f"{name}: fixed acc {fixed['acc']:.2f}, tuned acc {best['acc']:.2f} {best['params']}", flush=True)

    print("\n| Model | Thresholds | Stance accuracy | Macro-F1 | Flipped | Relevance P / R | Time per paragraph (CPU) |")
    print("| --- | --- | ---: | ---: | ---: | ---: | ---: |")
    for name, fixed, best, t in rows:
        for label, s in (("fixed 0.5/0.5/0.2", fixed), (f"tuned {best['params']}", best)):
            print(f"| {name} | {label} | {s['acc']:.2f} | {s['f1']:.2f} | {s['flips']} | {s['rel_p']:.2f} / {s['rel_r']:.2f} | {t:.2f} s |")
    if args.dump:
        Path(args.dump).write_text(json.dumps(dump, indent=1))


if __name__ == "__main__":
    main()
