-- Results of `kurswechsel eval`. Only a run against a reviewed gold set may be
-- published on the website, so the flag is set explicitly by whoever ran it.
CREATE TABLE evaluations (
    id                  INTEGER PRIMARY KEY,
    topic               TEXT NOT NULL,
    classifier          TEXT NOT NULL,
    prompt_version      TEXT NOT NULL,
    items               INTEGER NOT NULL,
    relevance_precision REAL NOT NULL,
    relevance_recall    REAL NOT NULL,
    stance_accuracy     REAL NOT NULL,
    stance_macro_f1     REAL NOT NULL,
    flip_rate           REAL NOT NULL,
    gold_reviewed       INTEGER NOT NULL,
    created_at          TEXT NOT NULL
);
