CREATE TABLE topics (
    slug     TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    question TEXT NOT NULL,
    keywords TEXT NOT NULL
);

CREATE TABLE paragraph_topics (
    paragraph_id INTEGER NOT NULL REFERENCES paragraphs(id) ON DELETE CASCADE,
    topic        TEXT NOT NULL REFERENCES topics(slug),
    PRIMARY KEY (paragraph_id, topic)
);
CREATE INDEX paragraph_topics_topic ON paragraph_topics(topic);

-- One classification per paragraph and topic. Model and prompt version are
-- kept so every entry on the website can be traced to what produced it.
CREATE TABLE stances (
    paragraph_id   INTEGER NOT NULL REFERENCES paragraphs(id) ON DELETE CASCADE,
    topic          TEXT NOT NULL REFERENCES topics(slug),
    relevant       INTEGER NOT NULL,
    stance         TEXT NOT NULL CHECK (stance IN ('dafuer', 'dagegen', 'neutral', 'unklar')),
    quote          TEXT NOT NULL,
    quote_verbatim INTEGER NOT NULL,
    rationale      TEXT NOT NULL,
    confidence     REAL NOT NULL,
    model          TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    classified_at  TEXT NOT NULL,
    PRIMARY KEY (paragraph_id, topic)
);
CREATE INDEX stances_topic ON stances(topic, relevant);
