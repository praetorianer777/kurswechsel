package stance

import (
	"strings"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Prompt styles. StyleFull asks the model for label, quote and rationale;
// StyleLabel asks for the label only and picks the quote by rule, which
// leaves small models a much simpler task (issue #22).
const (
	StyleFull  = "full"
	StyleLabel = "label"
)

// LabelPromptVersion identifies the label-only prompt in stored results.
const LabelPromptVersion = "v2-label"

// LabelSystemPrompt asks for relevance and stance only.
func LabelSystemPrompt(t topic.Topic) string {
	return `You classify paragraphs from speeches in the German Bundestag.

` + t.Definition + `

Answer with "relevant" and "stance" only. Decide from the paragraph alone and judge only the speaker's own position; do not use your knowledge of the speaker or their party.`
}

// LabelSchema constrains the label-only reply.
func LabelSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"relevant": map[string]any{"type": "boolean"},
			"stance":   map[string]any{"type": "string", "enum": Labels},
		},
		"required":             []string{"relevant", "stance"},
		"additionalProperties": false,
	}
}

// QuoteFor picks the first sentence of paragraph that contains one of the
// topic's keywords, or the whole paragraph if none does. The result always
// occurs verbatim in the paragraph.
func QuoteFor(paragraph string, t topic.Topic) string {
	for _, s := range Sentences(paragraph) {
		if t.Keywords.MatchString(s) {
			return s
		}
	}
	return strings.TrimSpace(paragraph)
}

// Sentences splits text after ".", "!" or "?" when a space and an upper-case
// letter or an opening quotation mark follow, so "Dr. Müller" and "z. B. die"
// stay together as often as a rule can manage.
func Sentences(text string) []string {
	var out []string
	runes := []rune(text)
	start := 0
	for i := 0; i < len(runes); i++ {
		if r := runes[i]; r != '.' && r != '!' && r != '?' {
			continue
		}
		j := i + 1
		for j < len(runes) && (runes[j] == '“' || runes[j] == '"' || runes[j] == ')') {
			j++
		}
		if j+1 < len(runes) && runes[j] == ' ' && startsSentence(runes[j+1]) && !abbreviation(runes[start:i+1]) {
			out = append(out, strings.TrimSpace(string(runes[start:j])))
			start = j + 1
		}
	}
	if rest := strings.TrimSpace(string(runes[start:])); rest != "" {
		out = append(out, rest)
	}
	return out
}

func startsSentence(r rune) bool {
	return r == '„' || r == '"' || r == '–' || (r >= 'A' && r <= 'Z') || strings.ContainsRune("ÄÖÜ", r)
}

var abbreviations = []string{"Dr.", "Prof.", "Nr.", "Abs.", "Art.", "bzw.", "ca.", "Mio.", "Mrd.", "vgl.", "St."}

// abbreviation reports whether the period ending sentence belongs to an
// abbreviation, a single letter ("z. B.") or an ordinal number ("1. Januar").
func abbreviation(sentence []rune) bool {
	fields := strings.Fields(string(sentence))
	if len(fields) == 0 {
		return false
	}
	last := strings.TrimSuffix(fields[len(fields)-1], ".")
	if len([]rune(last)) == 1 {
		return true
	}
	if strings.Trim(last, "0123456789") == "" {
		return true
	}
	for _, a := range abbreviations {
		if last+"." == a {
			return true
		}
	}
	return false
}
