package store

import (
	"context"
	"database/sql"
	"errors"
)

// Neighbour is a paragraph shown around the one being judged.
type Neighbour struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// ParagraphContext is what surrounds a paragraph: the speaker's own
// paragraphs before and after it, and, at the start of a speech, the end of
// the previous speech in the same session, which a short reply or a
// question often answers.
type ParagraphContext struct {
	Before   []Neighbour `json:"before"`
	After    []Neighbour `json:"after"`
	Previous *Neighbour  `json:"previous,omitempty"`
}

// Context returns up to before paragraphs before and after paragraphs after
// the given one in the same speech.
func (s *Store) Context(ctx context.Context, speechID string, position, before, after int) (ParagraphContext, error) {
	var pc ParagraphContext
	var speaker string
	var period, session int
	err := s.db.QueryRowContext(ctx, `SELECT speaker_name, period, session FROM speeches WHERE id = ?`, speechID).
		Scan(&speaker, &period, &session)
	if errors.Is(err, sql.ErrNoRows) {
		return pc, ErrNotFound
	}
	if err != nil {
		return pc, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT position, text FROM paragraphs
		WHERE speech_id = ? AND position BETWEEN ? AND ? AND position <> ?
		ORDER BY position`, speechID, position-before, position+after, position)
	if err != nil {
		return pc, err
	}
	for rows.Next() {
		var pos int
		var text string
		if err := rows.Scan(&pos, &text); err != nil {
			rows.Close()
			return pc, err
		}
		n := Neighbour{Speaker: speaker, Text: text}
		if pos < position {
			pc.Before = append(pc.Before, n)
		} else {
			pc.After = append(pc.After, n)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return pc, err
	}
	if position-before > 0 {
		return pc, nil
	}
	// Speech IDs grow in document order within a session.
	var prev Neighbour
	err = s.db.QueryRowContext(ctx, `
		SELECT sp.speaker_name, p.text
		FROM speeches sp JOIN paragraphs p ON p.speech_id = sp.id
		WHERE sp.period = ? AND sp.session = ? AND sp.id < ?
		ORDER BY sp.id DESC, p.position DESC LIMIT 1`, period, session, speechID).Scan(&prev.Speaker, &prev.Text)
	if errors.Is(err, sql.ErrNoRows) {
		return pc, nil
	}
	if err != nil {
		return pc, err
	}
	pc.Previous = &prev
	return pc, nil
}

// FindParagraph locates a paragraph by its exact text, for data whose own
// IDs do not match the database (the gold set was drawn with an earlier
// parser).
func (s *Store) FindParagraph(ctx context.Context, text string) (speechID string, position int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT speech_id, position FROM paragraphs WHERE text = ? LIMIT 1`, text).
		Scan(&speechID, &position)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, ErrNotFound
	}
	return speechID, position, err
}
