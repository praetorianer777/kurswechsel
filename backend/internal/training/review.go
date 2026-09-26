package training

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
)

// Reviewer serves a local page for checking pre-labels one at a time. Every
// verdict is written to the file at once, so the review can stop anywhere.
type Reviewer struct {
	Path string
	Now  func() time.Time
	// Context looks up the paragraphs around an item; nil shows none.
	Context func(speechID string, position int) (store.ParagraphContext, error)
	mu      sync.Mutex
	items   []Item
}

// NewReviewer loads the file and draws the agreement sample if needed.
func NewReviewer(path string, sample int) (*Reviewer, error) {
	items, err := Read(path)
	if err != nil {
		return nil, err
	}
	DrawSample(items, sample, 27)
	if err := Write(path, items); err != nil {
		return nil, err
	}
	return &Reviewer{Path: path, Now: time.Now, items: items}, nil
}

// Handler returns the review page's routes.
func (rv *Reviewer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rv.page)
	mux.HandleFunc("POST /review", rv.review)
	return mux
}

type view struct {
	Context  *store.ParagraphContext
	Stats    Stats
	Percent  string
	Item     *Item
	Marked   template.HTML
	Queue    []Item
	Labels   []string
	Names    map[string]string
	Relevant bool
	Stance   string
	Error    string
}

//go:embed review.html
var pageHTML string

var pageTmpl = template.Must(template.New("review").Parse(pageHTML))

var names = map[string]string{"dafuer": "dafür", "dagegen": "dagegen", "neutral": "neutral", "unklar": "unklar"}

func (rv *Reviewer) page(w http.ResponseWriter, r *http.Request) {
	rv.mu.Lock()
	defer rv.mu.Unlock()
	v := view{Stats: Summarise(rv.items), Labels: stance.Labels, Names: names}
	v.Percent = percent(v.Stats.Agreement())
	id := r.URL.Query().Get("id")
	for i := range rv.items {
		it := rv.items[i]
		if !it.NeedsReview() {
			continue
		}
		v.Queue = append(v.Queue, it)
		if v.Item == nil && ((id != "" && it.ID == id) || (id == "" && it.Review == nil)) {
			v.Item = &rv.items[i]
		}
	}
	if v.Item != nil {
		v.Relevant, v.Stance = v.Item.Final()
		v.Marked = mark(v.Item.Text, v.Item.Quote)
		if rv.Context != nil {
			if speech, pos, ok := splitID(v.Item.ID); ok {
				if c, err := rv.Context(speech, pos); err == nil {
					v.Context = &c
				}
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (rv *Reviewer) review(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	relevant := r.FormValue("relevant") == "ja"
	st := r.FormValue("stance")
	if !relevant {
		st = ""
	}
	if err := Validate(relevant, st); err != nil {
		http.Error(w, "Bitte eine Haltung wählen, wenn der Absatz relevant ist.", http.StatusUnprocessableEntity)
		return
	}
	rv.mu.Lock()
	defer rv.mu.Unlock()
	id := r.FormValue("id")
	for i := range rv.items {
		if rv.items[i].ID != id || !rv.items[i].NeedsReview() {
			continue
		}
		rv.items[i].Review = &Review{Relevant: relevant, Stance: st, Note: strings.TrimSpace(r.FormValue("note")), At: rv.Now().UTC()}
		if err := Write(rv.Path, rv.items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Error(w, "Unbekannter Eintrag", http.StatusNotFound)
}

// splitID reads "ID2000700100#3" into speech ID and paragraph position.
func splitID(id string) (string, int, bool) {
	speech, pos, ok := strings.Cut(id, "#")
	if !ok {
		return "", 0, false
	}
	n, err := strconv.Atoi(pos)
	return speech, n, err == nil
}

func percent(f float64) string { return fmt.Sprintf("%.0f %%", 100*f) }

// mark returns the escaped text with the quote highlighted.
func mark(text, quote string) template.HTML {
	esc := template.HTMLEscapeString(text)
	q := template.HTMLEscapeString(strings.TrimSpace(quote))
	if q == "" || !strings.Contains(esc, q) {
		return template.HTML(esc)
	}
	return template.HTML(strings.Replace(esc, q, "<mark>"+q+"</mark>", 1))
}
