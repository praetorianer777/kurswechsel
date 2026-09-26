package training

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func reviewer(t *testing.T) (*Reviewer, http.Handler) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "train.jsonl")
	items := labelled(6, 2)
	items[0].Text = "Die Wehrpflicht <muss> zurück."
	items[0].Quote = "Wehrpflicht <muss>"
	if err := Write(path, items); err != nil {
		t.Fatal(err)
	}
	rv, err := NewReviewer(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	return rv, rv.Handler()
}

func get(h http.Handler, path string) string {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec.Body.String()
}

func post(h http.Handler, form url.Values) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/review", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestReviewFlow(t *testing.T) {
	rv, h := reviewer(t)
	page := get(h, "/")
	if !strings.Contains(page, "0 von 4 geprüft") || !strings.Contains(page, "<mark>Wehrpflicht &lt;muss&gt;</mark>") {
		t.Fatalf("first page:\n%s", page)
	}
	if strings.Contains(page, "<muss>") {
		t.Error("paragraph text not escaped")
	}

	// Confirm the first sample item and correct the second.
	var sample []string
	for _, it := range rv.items {
		if it.Sampled {
			sample = append(sample, it.ID)
		}
	}
	if code := post(h, url.Values{"id": {sample[0]}, "relevant": {"ja"}, "stance": {"dafuer"}}); code != http.StatusSeeOther {
		t.Fatalf("confirm: %d", code)
	}
	if code := post(h, url.Values{"id": {sample[1]}, "relevant": {"nein"}, "stance": {"dafuer"}, "note": {" kein Thema "}}); code != http.StatusSeeOther {
		t.Fatalf("correct: %d", code)
	}
	items, err := Read(rv.Path)
	if err != nil {
		t.Fatal(err)
	}
	st := Summarise(items)
	if st.Reviewed != 2 || st.SampleAgreed != 1 || st.Agreement() != 0.5 {
		t.Errorf("stats after review = %+v", st)
	}
	for _, it := range items {
		if it.ID == sample[1] && (it.Review.Relevant || it.Review.Stance != "" || it.Review.Note != "kein Thema") {
			t.Errorf("correction stored as %+v", it.Review)
		}
	}
	if !strings.Contains(get(h, "/"), "50 % (1 von 2)") {
		t.Error("agreement not shown")
	}
	if !strings.Contains(get(h, "/?id="+sample[0]), "bereits geprüft") {
		t.Error("revisiting a reviewed item")
	}
}

func TestReviewRejects(t *testing.T) {
	_, h := reviewer(t)
	if code := post(h, url.Values{"id": {"a"}, "relevant": {"ja"}}); code != http.StatusUnprocessableEntity {
		t.Errorf("relevant without stance: %d", code)
	}
	if code := post(h, url.Values{"id": {"nope"}, "relevant": {"nein"}}); code != http.StatusNotFound {
		t.Errorf("unknown id: %d", code)
	}
}

func TestReviewDone(t *testing.T) {
	rv, h := reviewer(t)
	for _, it := range rv.items {
		if it.NeedsReview() {
			post(h, url.Values{"id": {it.ID}, "relevant": {"ja"}, "stance": {"dafuer"}})
		}
	}
	if !strings.Contains(get(h, "/"), "Alles geprüft.") {
		t.Error("finished page missing")
	}
}
