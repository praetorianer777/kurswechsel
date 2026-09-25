package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/timeline"
)

// ErrNotFound means the requested row does not exist.
var ErrNotFound = errors.New("not found")

// TopicSummary is a topic with how much the timeline has on it.
type TopicSummary struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Question string `json:"question"`
	People   int    `json:"people"`
	Entries  int    `json:"entries"`
}

// Topics lists all topics with counts of relevant classified paragraphs.
func (s *Store) Topics(ctx context.Context) ([]TopicSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.slug, t.name, t.question,
			(SELECT count(DISTINCT sp.politician_id) FROM stances st
				JOIN paragraphs p ON p.id = st.paragraph_id JOIN speeches sp ON sp.id = p.speech_id
				WHERE st.topic = t.slug AND st.relevant = 1),
			(SELECT count(*) FROM stances st WHERE st.topic = t.slug AND st.relevant = 1)
		FROM topics t ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopicSummary
	for rows.Next() {
		var t TopicSummary
		if err := rows.Scan(&t.Slug, &t.Name, &t.Question, &t.People, &t.Entries); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// PoliticianSummary is one person in search results.
type PoliticianSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Faction string `json:"faction"`
	Role    string `json:"role,omitempty"`
	// Entries counts relevant statements on the requested topic, or all
	// speeches when no topic was given.
	Entries int `json:"entries"`
}

// Politicians lists everyone who spoke, with their most recent faction or
// role. With a topic, only people with relevant statements on it are listed.
func (s *Store) Politicians(ctx context.Context, topicSlug string) ([]PoliticianSummary, error) {
	query := `
		WITH latest AS (
			SELECT politician_id, faction, role,
				row_number() OVER (PARTITION BY politician_id ORDER BY period DESC, session DESC) AS rn
			FROM speeches
		)
		SELECT p.id, p.title, p.first_name, p.last_name, p.affix, l.faction, l.role, count(*)
		FROM politicians p
		JOIN latest l ON l.politician_id = p.id AND l.rn = 1
		JOIN speeches sp ON sp.politician_id = p.id
		GROUP BY p.id
		ORDER BY p.last_name, p.first_name`
	args := []any{}
	if topicSlug != "" {
		query = `
		WITH latest AS (
			SELECT politician_id, faction, role,
				row_number() OVER (PARTITION BY politician_id ORDER BY period DESC, session DESC) AS rn
			FROM speeches
		)
		SELECT p.id, p.title, p.first_name, p.last_name, p.affix, l.faction, l.role, count(*)
		FROM politicians p
		JOIN latest l ON l.politician_id = p.id AND l.rn = 1
		JOIN speeches sp ON sp.politician_id = p.id
		JOIN paragraphs pa ON pa.speech_id = sp.id
		JOIN stances st ON st.paragraph_id = pa.id AND st.topic = ? AND st.relevant = 1
		GROUP BY p.id
		ORDER BY count(*) DESC, p.last_name, p.first_name`
		args = append(args, topicSlug)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PoliticianSummary
	for rows.Next() {
		var p PoliticianSummary
		var title, first, last, affix string
		if err := rows.Scan(&p.ID, &title, &first, &last, &affix, &p.Faction, &p.Role, &p.Entries); err != nil {
			return nil, err
		}
		p.Name = joinName(title, first, last, affix)
		out = append(out, p)
	}
	return out, rows.Err()
}

func joinName(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// FactionSpan is a faction membership from the master data.
type FactionSpan struct {
	Period  int    `json:"period"`
	Faction string `json:"faction"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

// TopicCount is how many relevant statements a person has on a topic.
type TopicCount struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Entries int    `json:"entries"`
}

// PoliticianDetail is one person's profile.
type PoliticianDetail struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Party    string        `json:"party,omitempty"`
	IsMdB    bool          `json:"is_mdb"`
	Factions []FactionSpan `json:"factions"`
	Topics   []TopicCount  `json:"topics"`
}

// Politician returns one person's profile or ErrNotFound.
func (s *Store) Politician(ctx context.Context, id string) (PoliticianDetail, error) {
	var d PoliticianDetail
	var title, first, last, affix string
	err := s.db.QueryRowContext(ctx, `SELECT id, title, first_name, last_name, affix, party, is_mdb FROM politicians WHERE id = ?`, id).
		Scan(&d.ID, &title, &first, &last, &affix, &d.Party, &d.IsMdB)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	d.Name = joinName(title, first, last, affix)

	d.Factions = []FactionSpan{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT period, faction, coalesce(valid_from, ''), coalesce(valid_to, '')
		FROM memberships WHERE politician_id = ? ORDER BY period, valid_from`, id)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var f FactionSpan
		if err := rows.Scan(&f.Period, &f.Faction, &f.From, &f.To); err != nil {
			rows.Close()
			return d, err
		}
		d.Factions = append(d.Factions, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return d, err
	}

	d.Topics = []TopicCount{}
	rows, err = s.db.QueryContext(ctx, `
		SELECT t.slug, t.name, count(*)
		FROM stances st
		JOIN topics t ON t.slug = st.topic
		JOIN paragraphs p ON p.id = st.paragraph_id
		JOIN speeches sp ON sp.id = p.speech_id
		WHERE sp.politician_id = ? AND st.relevant = 1
		GROUP BY t.slug ORDER BY t.name`, id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var t TopicCount
		if err := rows.Scan(&t.Slug, &t.Name, &t.Entries); err != nil {
			return d, err
		}
		d.Topics = append(d.Topics, t)
	}
	return d, rows.Err()
}

// Timeline returns a person's relevant statements on a topic, unsorted and
// without change marks; timeline.MarkChanges does both.
func (s *Store) Timeline(ctx context.Context, politicianID, topicSlug string) ([]timeline.Entry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, sp.id, p.position, se.date, sp.period, sp.session, coalesce(p.page, sp.page, 0),
			sp.faction, sp.role, p.text, st.stance, st.quote, st.quote_verbatim, st.rationale,
			st.confidence, st.model, se.source_url
		FROM stances st
		JOIN paragraphs p ON p.id = st.paragraph_id
		JOIN speeches sp ON sp.id = p.speech_id
		JOIN sessions se ON se.period = sp.period AND se.number = sp.session
		WHERE sp.politician_id = ? AND st.topic = ? AND st.relevant = 1`, politicianID, topicSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []timeline.Entry{}
	for rows.Next() {
		var e timeline.Entry
		var date string
		var verbatim bool
		if err := rows.Scan(&e.ParagraphID, &e.SpeechID, &e.Position, &date, &e.Period, &e.Session, &e.Page,
			&e.Faction, &e.Role, &e.Text, &e.Stance, &e.Quote, &verbatim, &e.Rationale,
			&e.Confidence, &e.Model, &e.SourceURL); err != nil {
			return nil, err
		}
		if e.Date, err = time.Parse(dateFormat, date[:10]); err != nil {
			return nil, err
		}
		if !verbatim {
			e.Quote = ""
		}
		// The XML is the machine-readable record; people want the PDF.
		e.SourceURL = strings.TrimSuffix(e.SourceURL, ".xml") + ".pdf"
		out = append(out, e)
	}
	return out, rows.Err()
}

// Report is an error report from the website.
type Report struct {
	ParagraphID int64
	Topic       string
	Message     string
	Contact     string
}

// SaveReport stores a report. It fails if the paragraph or topic does not
// exist.
func (s *Store) SaveReport(ctx context.Context, r Report, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO reports (paragraph_id, topic, message, contact, created_at) VALUES (?, ?, ?, ?, ?)`,
		r.ParagraphID, r.Topic, r.Message, r.Contact, now.UTC().Format(time.RFC3339))
	return err
}
