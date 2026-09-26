package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// SyncTopics stores the topic definitions.
func (s *Store) SyncTopics(ctx context.Context, ts []topic.Topic) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		for _, t := range ts {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO topics (slug, name, question, keywords) VALUES (?, ?, ?, ?)
				ON CONFLICT (slug) DO UPDATE SET name = excluded.name, question = excluded.question, keywords = excluded.keywords`,
				t.Slug, t.Name, t.Question, t.Keywords.String()); err != nil {
				return err
			}
		}
		return nil
	})
}

// AssignTopic marks every paragraph matching the topic's keywords as a
// candidate and returns how many there are. Quotations are skipped: they are
// someone else's words and must not end up on the speaker's timeline.
func (s *Store) AssignTopic(ctx context.Context, t topic.Topic) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, text FROM paragraphs WHERE is_quote = 0`)
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			rows.Close()
			return 0, err
		}
		if t.Keywords.MatchString(text) {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	err = s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM paragraph_topics WHERE topic = ?`, t.Slug); err != nil {
			return err
		}
		ins, err := tx.PrepareContext(ctx, `INSERT INTO paragraph_topics (paragraph_id, topic) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := ins.ExecContext(ctx, id, t.Slug); err != nil {
				return err
			}
		}
		return nil
	})
	return len(ids), err
}

// Candidate is a paragraph waiting for classification.
type Candidate struct {
	ParagraphID int64
	Text        string
}

// Pending returns candidates of a topic that have no stance yet, or one from
// an older prompt version, oldest session first. limit <= 0 means all.
func (s *Store) Pending(ctx context.Context, slug string, limit int) ([]Candidate, error) {
	if limit <= 0 {
		limit = -1
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.text
		FROM paragraph_topics pt
		JOIN paragraphs p ON p.id = pt.paragraph_id
		JOIN speeches sp ON sp.id = p.speech_id
		JOIN sessions se ON se.period = sp.period AND se.number = sp.session
		LEFT JOIN stances st ON st.paragraph_id = p.id AND st.topic = pt.topic
		WHERE pt.topic = ? AND (st.paragraph_id IS NULL OR st.prompt_version <> ?)
		ORDER BY se.date, sp.id, p.position
		LIMIT ?`, slug, stance.PromptVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.ParagraphID, &c.Text); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SaveStance stores a classification, replacing an earlier one.
func (s *Store) SaveStance(ctx context.Context, paragraphID int64, slug string, a stance.Answer, verbatim bool, model string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stances (paragraph_id, topic, relevant, stance, quote, quote_verbatim, rationale, confidence, model, prompt_version, classified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (paragraph_id, topic) DO UPDATE SET
			relevant = excluded.relevant, stance = excluded.stance, quote = excluded.quote,
			quote_verbatim = excluded.quote_verbatim, rationale = excluded.rationale, confidence = excluded.confidence,
			model = excluded.model, prompt_version = excluded.prompt_version, classified_at = excluded.classified_at`,
		paragraphID, slug, a.Relevant, a.Stance, a.Quote, verbatim, a.Rationale, a.Confidence, model, stance.PromptVersion,
		now.UTC().Format(time.RFC3339))
	return err
}

// DeleteStances removes all classifications of a topic, so the next run
// classifies everything again.
func (s *Store) DeleteStances(ctx context.Context, slug string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM stances WHERE topic = ?`, slug)
	return err
}
