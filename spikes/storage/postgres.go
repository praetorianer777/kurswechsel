package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/praetorianer777/kurswechsel/spikes/corpus"
)

// Postgres uses a generated tsvector column with the german text search
// configuration and a GIN index. Open needs a URL, not a directory.
type Postgres struct {
	URL string
	db  *sql.DB
}

func (*Postgres) Name() string { return "PostgreSQL (tsvector german, GIN)" }

func (p *Postgres) Open(ctx context.Context, _ string) error {
	db, err := sql.Open("pgx", p.URL)
	if err != nil {
		return err
	}
	p.db = db
	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS paragraphs, speeches;
		CREATE TABLE speeches (id TEXT PRIMARY KEY, date DATE NOT NULL, speaker_id TEXT NOT NULL, speaker TEXT, party TEXT);
		CREATE INDEX speeches_speaker ON speeches(speaker_id, date);
		CREATE TABLE paragraphs (
			id BIGSERIAL PRIMARY KEY,
			speech_id TEXT NOT NULL REFERENCES speeches(id),
			pos INTEGER NOT NULL,
			text TEXT NOT NULL,
			tsv tsvector GENERATED ALWAYS AS (to_tsvector('german', text)) STORED);
		CREATE INDEX paragraphs_speech ON paragraphs(speech_id);`)
	return err
}

func (p *Postgres) Load(ctx context.Context, speeches []corpus.Speech) error {
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(dc any) error {
		pc := dc.(*stdlib.Conn).Conn()
		srows := make([][]any, 0, len(speeches))
		var prows [][]any
		for _, sp := range speeches {
			srows = append(srows, []any{sp.ID, sp.Date, sp.SpeakerID, sp.Speaker, sp.Party})
			for i, t := range sp.Paragraphs {
				prows = append(prows, []any{sp.ID, i, t})
			}
		}
		if _, err := pc.CopyFrom(ctx, pgx.Identifier{"speeches"}, []string{"id", "date", "speaker_id", "speaker", "party"}, pgx.CopyFromRows(srows)); err != nil {
			return err
		}
		if _, err := pc.CopyFrom(ctx, pgx.Identifier{"paragraphs"}, []string{"speech_id", "pos", "text"}, pgx.CopyFromRows(prows)); err != nil {
			return err
		}
		_, err := pc.Exec(ctx, `CREATE INDEX paragraphs_tsv ON paragraphs USING gin(tsv); ANALYZE;`)
		return err
	})
}

func tsQuery(terms []string) string {
	q := make([]string, len(terms))
	for i, t := range terms {
		q[i] = t + ":*"
	}
	return strings.Join(q, " | ")
}

func (p *Postgres) Search(ctx context.Context, terms []string) (int, error) {
	var n int
	err := p.db.QueryRowContext(ctx, `SELECT count(*) FROM paragraphs WHERE tsv @@ to_tsquery('german', $1)`, tsQuery(terms)).Scan(&n)
	return n, err
}

func (p *Postgres) Timeline(ctx context.Context, speakerID string, terms []string) ([]Hit, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT s.date, s.id, p.text
		FROM paragraphs p JOIN speeches s ON s.id = p.speech_id
		WHERE s.speaker_id = $1 AND p.tsv @@ to_tsquery('german', $2)
		ORDER BY s.date, s.id, p.pos`, speakerID, tsQuery(terms))
	if err != nil {
		return nil, err
	}
	return scanHits(rows)
}

func (p *Postgres) Size(ctx context.Context) (int64, error) {
	var n int64
	err := p.db.QueryRowContext(ctx, `SELECT pg_total_relation_size('speeches') + pg_total_relation_size('paragraphs')`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("relation size: %w", err)
	}
	return n, nil
}

func (p *Postgres) Close() error { return p.db.Close() }
