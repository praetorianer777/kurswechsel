// Package topic defines the topics Kurswechsel tracks.
package topic

import "regexp"

// Topic is one political question the timeline follows.
type Topic struct {
	Slug string
	// Name and Question are shown on the website, so they are German.
	Name     string
	Question string
	// Definition tells the classifier what counts as relevant and what each
	// stance means for this topic.
	Definition string
	// Keywords preselects candidate paragraphs (ADR 0003).
	Keywords *regexp.Regexp
}

// Wehrpflicht is the first topic, chosen because many positions shifted
// between the 2011 suspension and the 2025 military service law.
var Wehrpflicht = Topic{
	Slug:     "wehrpflicht",
	Name:     "Wehrpflicht",
	Question: "Soll der Wehrdienst oder ein allgemeiner Dienst in Deutschland verpflichtend sein?",
	Definition: `Topic: whether military service, or a general duty to serve that includes it, should be COMPULSORY in Germany (Wehrpflicht, verpflichtender Wehrdienst, Musterungspflicht, allgemeine Dienstpflicht).

- relevant: true only if the paragraph substantively discusses that question, including the 2025 military service law. A passing mention (history, other countries, compensation for service-related injuries) is not relevant.
- stance:
  - "dafuer": the SPEAKER supports compulsion: reinstating conscription, compulsory screening of cohorts, a mandatory year of service, or criticises its suspension.
  - "dagegen": the SPEAKER opposes compulsion or insists on voluntary service only.
  - "neutral": facts, procedure or a model without taking a side on compulsion.
  - "unklar": irony, rhetorical questions, or other people's views; a position cannot be read off reliably.`,
	Keywords: regexp.MustCompile(`(?i)wehrpflicht|wehrdienst|musterung|dienstpflicht|pflichtdienst|gesellschaftsjahr|pflichtjahr|dienst an der waffe`),
}

// All lists every topic.
var All = []Topic{Wehrpflicht}

// Get finds a topic by slug.
func Get(slug string) (Topic, bool) {
	for _, t := range All {
		if t.Slug == slug {
			return t, true
		}
	}
	return Topic{}, false
}
