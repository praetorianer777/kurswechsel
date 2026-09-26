//go:build duckdb

package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"strings"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/praetorianer777/kurswechsel/spikes/corpus"
)

func init() { extraBackends = append(extraBackends, func() Backend { return &DuckDB{} }) }

// DuckDB uses the cgo driver with its appender for loading. The fts extension
// would have to be downloaded at runtime, so search is an ILIKE scan, which is
// what a columnar engine is built for anyway.
type DuckDB struct {
	connector *duckdb.Connector
	db        *sql.DB
	dir       string
}

func (*DuckDB) Name() string { return "DuckDB (cgo, ILIKE scan)" }

func (d *DuckDB) Open(ctx context.Context, dir string) error {
	d.dir = dir
	c, err := duckdb.NewConnector(filepath.Join(dir, "bench.duckdb"), nil)
	if err != nil {
		return err
	}
	d.connector = c
	d.db = sql.OpenDB(c)
	_, err = d.db.ExecContext(ctx, `
		CREATE TABLE speeches (id VARCHAR PRIMARY KEY, date DATE NOT NULL, speaker_id VARCHAR NOT NULL, speaker VARCHAR, party VARCHAR);
		CREATE TABLE paragraphs (speech_id VARCHAR NOT NULL, pos INTEGER NOT NULL, text VARCHAR NOT NULL);`)
	return err
}

func (d *DuckDB) Load(ctx context.Context, speeches []corpus.Speech) error {
	conn, err := d.connector.Connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	as, err := duckdb.NewAppenderFromConn(conn, "", "speeches")
	if err != nil {
		return err
	}
	ap, err := duckdb.NewAppenderFromConn(conn, "", "paragraphs")
	if err != nil {
		return err
	}
	for _, sp := range speeches {
		if err := as.AppendRow(sp.ID, sp.Date, sp.SpeakerID, sp.Speaker, sp.Party); err != nil {
			return err
		}
		for i, p := range sp.Paragraphs {
			if err := ap.AppendRow(sp.ID, int32(i), p); err != nil {
				return err
			}
		}
	}
	if err := as.Close(); err != nil {
		return err
	}
	return ap.Close()
}

func ilike(terms []string) (string, []driver.Value) {
	conds := make([]string, len(terms))
	args := make([]driver.Value, len(terms))
	for i, t := range terms {
		conds[i] = "p.text ILIKE ?"
		args[i] = "%" + t + "%"
	}
	return "(" + strings.Join(conds, " OR ") + ")", args
}

func toAny(vs []driver.Value) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}

func (d *DuckDB) Search(ctx context.Context, terms []string) (int, error) {
	cond, args := ilike(terms)
	var n int
	err := d.db.QueryRowContext(ctx, `SELECT count(*) FROM paragraphs p WHERE `+cond, toAny(args)...).Scan(&n)
	return n, err
}

func (d *DuckDB) Timeline(ctx context.Context, speakerID string, terms []string) ([]Hit, error) {
	cond, args := ilike(terms)
	rows, err := d.db.QueryContext(ctx, `
		SELECT s.date, s.id, p.text
		FROM paragraphs p JOIN speeches s ON s.id = p.speech_id
		WHERE s.speaker_id = ? AND `+cond+`
		ORDER BY s.date, s.id, p.pos`, append([]any{speakerID}, toAny(args)...)...)
	if err != nil {
		return nil, err
	}
	return scanHits(rows)
}

func (d *DuckDB) Size(ctx context.Context) (int64, error) {
	if _, err := d.db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return 0, err
	}
	return dirSize(d.dir)
}

func (d *DuckDB) Close() error {
	err := d.db.Close()
	if cerr := d.connector.Close(); err == nil {
		err = cerr
	}
	return err
}
