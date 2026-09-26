CREATE TABLE politicians (
    id         TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    affix      TEXT NOT NULL DEFAULT '',
    party      TEXT NOT NULL DEFAULT '',
    gender     TEXT NOT NULL DEFAULT '',
    -- 0 for speakers who appear in protocols but not in the MdB master data,
    -- such as members of the government without a seat.
    is_mdb     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE memberships (
    politician_id TEXT NOT NULL REFERENCES politicians(id) ON DELETE CASCADE,
    period        INTEGER NOT NULL,
    faction       TEXT NOT NULL,
    valid_from    DATE,
    valid_to      DATE,
    PRIMARY KEY (politician_id, period, faction, valid_from)
);

CREATE TABLE sessions (
    period      INTEGER NOT NULL,
    number      INTEGER NOT NULL,
    date        DATE NOT NULL,
    source_url  TEXT NOT NULL,
    ingested_at TEXT NOT NULL,
    PRIMARY KEY (period, number)
);

CREATE TABLE speeches (
    id            TEXT PRIMARY KEY,
    period        INTEGER NOT NULL,
    session       INTEGER NOT NULL,
    politician_id TEXT NOT NULL REFERENCES politicians(id),
    speaker_name  TEXT NOT NULL,
    faction       TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT '',
    page          INTEGER,
    FOREIGN KEY (period, session) REFERENCES sessions(period, number)
);
CREATE INDEX speeches_politician ON speeches(politician_id);
CREATE INDEX speeches_session ON speeches(period, session);

CREATE TABLE paragraphs (
    id        INTEGER PRIMARY KEY,
    speech_id TEXT NOT NULL REFERENCES speeches(id),
    position  INTEGER NOT NULL,
    text      TEXT NOT NULL,
    is_quote  INTEGER NOT NULL DEFAULT 0,
    page      INTEGER,
    UNIQUE (speech_id, position)
);

CREATE VIRTUAL TABLE paragraphs_fts USING fts5(
    text,
    content = 'paragraphs',
    content_rowid = 'id',
    tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER paragraphs_ai AFTER INSERT ON paragraphs BEGIN
    INSERT INTO paragraphs_fts(rowid, text) VALUES (new.id, new.text);
END;
CREATE TRIGGER paragraphs_ad AFTER DELETE ON paragraphs BEGIN
    INSERT INTO paragraphs_fts(paragraphs_fts, rowid, text) VALUES ('delete', old.id, old.text);
END;
CREATE TRIGGER paragraphs_au AFTER UPDATE OF text ON paragraphs BEGIN
    INSERT INTO paragraphs_fts(paragraphs_fts, rowid, text) VALUES ('delete', old.id, old.text);
    INSERT INTO paragraphs_fts(rowid, text) VALUES (new.id, new.text);
END;
