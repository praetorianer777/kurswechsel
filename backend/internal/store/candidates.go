package store

import (
	"context"
	"fmt"
)

// CandidateInfo is a topic candidate with the context a labeller needs.
type CandidateInfo struct {
	ID       string // speech ID and position, as in the gold set: "ID…#3"
	SpeechID string
	Date     string
	Period   int
	Speaker  string
	Faction  string
	Text     string
}

// Candidates lists all paragraphs assigned to a topic.
func (s *Store) Candidates(ctx context.Context, slug string) ([]CandidateInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sp.id, p.position, se.date, sp.period, sp.speaker_name,
			CASE WHEN sp.faction <> '' THEN sp.faction ELSE sp.role END, p.text
		FROM paragraph_topics pt
		JOIN paragraphs p ON p.id = pt.paragraph_id
		JOIN speeches sp ON sp.id = p.speech_id
		JOIN sessions se ON se.period = sp.period AND se.number = sp.session
		WHERE pt.topic = ?
		ORDER BY se.date, sp.id, p.position`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CandidateInfo
	for rows.Next() {
		var c CandidateInfo
		var pos int
		if err := rows.Scan(&c.SpeechID, &pos, &c.Date, &c.Period, &c.Speaker, &c.Faction, &c.Text); err != nil {
			return nil, err
		}
		c.ID = fmt.Sprintf("%s#%d", c.SpeechID, pos)
		c.Date = c.Date[:10]
		out = append(out, c)
	}
	return out, rows.Err()
}
