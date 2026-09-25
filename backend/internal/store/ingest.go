package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
)

const dateFormat = "2006-01-02"

func nullDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(dateFormat)
}

func nullInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

// UpsertPoliticians stores the MdB master data. Memberships are replaced, so
// corrections in the master data take effect.
func (s *Store) UpsertPoliticians(ctx context.Context, ps []bundestag.Politician) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		up, err := tx.PrepareContext(ctx, `
			INSERT INTO politicians (id, first_name, last_name, title, affix, party, gender, is_mdb)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1)
			ON CONFLICT (id) DO UPDATE SET
				first_name = excluded.first_name, last_name = excluded.last_name, title = excluded.title,
				affix = excluded.affix, party = excluded.party, gender = excluded.gender, is_mdb = 1`)
		if err != nil {
			return err
		}
		del, err := tx.PrepareContext(ctx, `DELETE FROM memberships WHERE politician_id = ?`)
		if err != nil {
			return err
		}
		ins, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO memberships (politician_id, period, faction, valid_from, valid_to) VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		for _, p := range ps {
			if _, err := up.ExecContext(ctx, p.ID, p.First, p.Last, p.Title, p.Affix, p.Party, p.Gender); err != nil {
				return err
			}
			if _, err := del.ExecContext(ctx, p.ID); err != nil {
				return err
			}
			for _, m := range p.Memberships {
				if _, err := ins.ExecContext(ctx, p.ID, m.Period, m.Faction, nullDate(m.From), nullDate(m.To)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// HasSession reports whether a session has been ingested.
func (s *Store) HasSession(ctx context.Context, period, number int) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sessions WHERE period = ? AND number = ?`, period, number).Scan(&n)
	return n > 0, err
}

// SaveProtocol stores one session with all its speeches. Saving the same
// session again updates it in place: paragraphs keep their IDs, so anything
// that refers to them (stances, reports) survives a re-import.
func (s *Store) SaveProtocol(ctx context.Context, p bundestag.Protocol, sourceURL string, now time.Time) error {
	date := p.Date.Format(dateFormat)
	return s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (period, number, date, source_url, ingested_at) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (period, number) DO UPDATE SET date = excluded.date, source_url = excluded.source_url, ingested_at = excluded.ingested_at`,
			p.Period, p.Session, date, sourceURL, now.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		speaker, err := tx.PrepareContext(ctx, `
			INSERT INTO politicians (id, first_name, last_name, title, affix) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (id) DO NOTHING`)
		if err != nil {
			return err
		}
		// A faction the protocol spells unrecognisably is taken from the
		// master data for the day of the session.
		speech, err := tx.PrepareContext(ctx, `
			INSERT INTO speeches (id, period, session, politician_id, speaker_name, faction, role, page)
			VALUES (?1, ?2, ?3, ?4, ?5,
				coalesce(nullif(?6, ''), (
					SELECT faction FROM memberships
					WHERE politician_id = ?4 AND (valid_from IS NULL OR valid_from <= ?9) AND (valid_to IS NULL OR valid_to >= ?9)
					ORDER BY valid_from DESC LIMIT 1), ''),
				?7, ?8)
			ON CONFLICT (id) DO UPDATE SET
				politician_id = excluded.politician_id, speaker_name = excluded.speaker_name,
				faction = excluded.faction, role = excluded.role, page = excluded.page`)
		if err != nil {
			return err
		}
		para, err := tx.PrepareContext(ctx, `
			INSERT INTO paragraphs (speech_id, position, text, is_quote, page) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (speech_id, position) DO UPDATE SET
				text = excluded.text, is_quote = excluded.is_quote, page = excluded.page
			WHERE text IS NOT excluded.text OR is_quote IS NOT excluded.is_quote OR page IS NOT excluded.page`)
		if err != nil {
			return err
		}
		trim, err := tx.PrepareContext(ctx, `DELETE FROM paragraphs WHERE speech_id = ? AND position >= ?`)
		if err != nil {
			return err
		}
		for _, sp := range p.Speeches {
			sk := sp.Speaker
			if _, err := speaker.ExecContext(ctx, sk.ID, sk.First, sk.Last, sk.Title, sk.Suffix); err != nil {
				return err
			}
			if _, err := speech.ExecContext(ctx, sp.ID, p.Period, p.Session, sk.ID, sk.Name(),
				bundestag.NormalizeFaction(sk.Faction), sk.Role, nullInt(sp.Page), date); err != nil {
				return err
			}
			for i, pg := range sp.Paragraphs {
				if _, err := para.ExecContext(ctx, sp.ID, i, pg.Text, pg.Quote, nullInt(pg.Page)); err != nil {
					return err
				}
			}
			if _, err := trim.ExecContext(ctx, sp.ID, len(sp.Paragraphs)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Stats counts what the database holds.
type Stats struct {
	Politicians, Sessions, Speeches, Paragraphs int
}

// Stats returns row counts.
func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM politicians), (SELECT count(*) FROM sessions),
		(SELECT count(*) FROM speeches), (SELECT count(*) FROM paragraphs)`).
		Scan(&st.Politicians, &st.Sessions, &st.Speeches, &st.Paragraphs)
	return st, err
}
