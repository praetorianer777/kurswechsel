package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
	_ "modernc.org/sqlite"
)

// SQLite is the pure-Go modernc driver with an FTS5 index.
type SQLite struct {
	db  *sql.DB
	dir string
}

func (*SQLite) Name() string { return "SQLite (modernc, FTS5)" }

func (s *SQLite) Open(ctx context.Context, dir string) error {
	s.dir = dir
	db, err := sql.Open("sqlite", filepath.Join(dir, "bench.sqlite")+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return err
	}
	s.db = db
	_, err = db.ExecContext(ctx, `
		CREATE TABLE speeches (id TEXT PRIMARY KEY, date DATE NOT NULL, speaker_id TEXT NOT NULL, speaker TEXT, party TEXT);
		CREATE INDEX speeches_speaker ON speeches(speaker_id, date);
		CREATE TABLE paragraphs (id INTEGER PRIMARY KEY, speech_id TEXT NOT NULL REFERENCES speeches(id), pos INTEGER NOT NULL, text TEXT NOT NULL);
		CREATE INDEX paragraphs_speech ON paragraphs(speech_id);
		CREATE VIRTUAL TABLE paragraphs_fts USING fts5(text, content='paragraphs', content_rowid='id', tokenize='unicode61 remove_diacritics 2');`)
	return err
}

func (s *SQLite) Load(ctx context.Context, speeches []corpus.Speech) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	insS, err := tx.PrepareContext(ctx, `INSERT INTO speeches VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	insP, err := tx.PrepareContext(ctx, `INSERT INTO paragraphs (speech_id, pos, text) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, sp := range speeches {
		if _, err := insS.ExecContext(ctx, sp.ID, sp.Date.Format("2006-01-02"), sp.SpeakerID, sp.Speaker, sp.Party); err != nil {
			return err
		}
		for i, p := range sp.Paragraphs {
			if _, err := insP.ExecContext(ctx, sp.ID, i, p); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO paragraphs_fts(paragraphs_fts) VALUES ('rebuild')`); err != nil {
		return err
	}
	return tx.Commit()
}

func ftsQuery(terms []string) string {
	q := make([]string, len(terms))
	for i, t := range terms {
		q[i] = t + "*"
	}
	return strings.Join(q, " OR ")
}

func (s *SQLite) Search(ctx context.Context, terms []string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM paragraphs_fts WHERE paragraphs_fts MATCH ?`, ftsQuery(terms)).Scan(&n)
	return n, err
}

func (s *SQLite) Timeline(ctx context.Context, speakerID string, terms []string) ([]Hit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.date, s.id, p.text
		FROM paragraphs_fts f
		JOIN paragraphs p ON p.id = f.rowid
		JOIN speeches s ON s.id = p.speech_id
		WHERE f.paragraphs_fts MATCH ? AND s.speaker_id = ?
		ORDER BY s.date, s.id, p.pos`, ftsQuery(terms), speakerID)
	if err != nil {
		return nil, err
	}
	return scanHits(rows)
}

func (s *SQLite) Size(ctx context.Context) (int64, error) {
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return 0, err
	}
	return dirSize(s.dir)
}

func (s *SQLite) Close() error { return s.db.Close() }
