-- Error reports from the website ("Fehler melden"). Contact is optional and
-- only kept so the report can be answered.
CREATE TABLE reports (
    id           INTEGER PRIMARY KEY,
    paragraph_id INTEGER NOT NULL REFERENCES paragraphs(id),
    topic        TEXT NOT NULL REFERENCES topics(slug),
    message      TEXT NOT NULL,
    contact      TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'rejected'))
);
