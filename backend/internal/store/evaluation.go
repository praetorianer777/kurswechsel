package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Evaluation is a stored summary of an eval run.
type Evaluation struct {
	Topic              string    `json:"topic"`
	Classifier         string    `json:"classifier"`
	PromptVersion      string    `json:"prompt_version"`
	Items              int       `json:"items"`
	RelevancePrecision float64   `json:"relevance_precision"`
	RelevanceRecall    float64   `json:"relevance_recall"`
	StanceAccuracy     float64   `json:"stance_accuracy"`
	StanceMacroF1      float64   `json:"stance_macro_f1"`
	FlipRate           float64   `json:"flip_rate"`
	GoldReviewed       bool      `json:"gold_reviewed"`
	CreatedAt          time.Time `json:"created_at"`
}

// SaveEvaluation stores an eval run.
func (s *Store) SaveEvaluation(ctx context.Context, e Evaluation) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO evaluations (topic, classifier, prompt_version, items, relevance_precision, relevance_recall,
			stance_accuracy, stance_macro_f1, flip_rate, gold_reviewed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Topic, e.Classifier, e.PromptVersion, e.Items, e.RelevancePrecision, e.RelevanceRecall,
		e.StanceAccuracy, e.StanceMacroF1, e.FlipRate, e.GoldReviewed, e.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// PublishedEvaluation returns the newest run against a reviewed gold set for
// the classifier that produced the topic's stances, or ErrNotFound.
func (s *Store) PublishedEvaluation(ctx context.Context, topic string) (Evaluation, error) {
	var e Evaluation
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT topic, classifier, prompt_version, items, relevance_precision, relevance_recall,
			stance_accuracy, stance_macro_f1, flip_rate, gold_reviewed, created_at
		FROM evaluations ev
		WHERE topic = ? AND gold_reviewed = 1
			AND classifier = (SELECT model FROM stances WHERE topic = ev.topic GROUP BY model ORDER BY count(*) DESC LIMIT 1)
			AND prompt_version = (SELECT prompt_version FROM stances WHERE topic = ev.topic GROUP BY prompt_version ORDER BY count(*) DESC LIMIT 1)
		ORDER BY created_at DESC, id DESC LIMIT 1`, topic).
		Scan(&e.Topic, &e.Classifier, &e.PromptVersion, &e.Items, &e.RelevancePrecision, &e.RelevanceRecall,
			&e.StanceAccuracy, &e.StanceMacroF1, &e.FlipRate, &e.GoldReviewed, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, err
	}
	e.CreatedAt, err = time.Parse(time.RFC3339, created)
	return e, err
}
