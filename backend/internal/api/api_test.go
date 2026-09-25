package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

var ctx = context.Background()

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// seed stores one person who is for conscription in 2019 and against it in
// 2024, plus a second person without statements on the topic.
func seed(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.UpsertPoliticians(ctx, []bundestag.Politician{{
		ID: "11000001", First: "Jürgen", Last: "Müller", Party: "CDU",
		Memberships: []bundestag.Membership{{Period: 19, Faction: bundestag.CDUCSU, From: day(2017, 10, 24), To: day(2021, 10, 26)}},
	}}); err != nil {
		t.Fatal(err)
	}
	protos := []bundestag.Protocol{
		{Period: 19, Session: 50, Date: day(2019, 3, 1), Speeches: []bundestag.Speech{{
			ID: "ID1", Page: 5500, Speaker: bundestag.Speaker{ID: "11000001", First: "Jürgen", Last: "Müller", Faction: "CDU/CSU"},
			Paragraphs: []bundestag.Paragraph{{Text: "Wir müssen die Wehrpflicht wieder einführen."}},
		}}},
		{Period: 20, Session: 90, Date: day(2024, 3, 1), Speeches: []bundestag.Speech{
			{
				ID: "ID2", Speaker: bundestag.Speaker{ID: "11000001", First: "Jürgen", Last: "Müller", Faction: "CDU/CSU"},
				Paragraphs: []bundestag.Paragraph{{Text: "Der Wehrdienst muss freiwillig bleiben."}, {Text: "Die Bundeswehr braucht Panzer."}},
			},
			{
				ID: "ID3", Speaker: bundestag.Speaker{ID: "99000001", First: "Erika", Last: "Ohnethema", Faction: "SPD"},
				Paragraphs: []bundestag.Paragraph{{Text: "Heute geht es um die Rente."}},
			},
		}},
	}
	for _, p := range protos {
		if err := st.SaveProtocol(ctx, p, "https://dserver.bundestag.de/btp/19/19050.xml", time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.SyncTopics(ctx, topic.All); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AssignTopic(ctx, topic.Wehrpflicht); err != nil {
		t.Fatal(err)
	}
	pending, err := st.Pending(ctx, "wehrpflicht", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range pending {
		a, _ := stance.Fake{}.Classify(ctx, topic.Wehrpflicht, c.Text)
		if err := st.SaveStance(ctx, c.ParagraphID, "wehrpflicht", a, stance.Verbatim(c.Text, a.Quote), "fake", time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	return st
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return NewHandler(&Server{
		Store:    seed(t),
		Frontend: fstest.MapFS{"index.html": {Data: []byte("<html>app</html>")}},
	})
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestHealthz(t *testing.T) {
	rec := do(newTestHandler(t), "GET", "/healthz", "")
	if rec.Code != 200 || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("%d %q", rec.Code, rec.Body.String())
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := do(newTestHandler(t), "GET", "/", "")
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
	if rec.Body.String() != "<html>app</html>" {
		t.Errorf("frontend not served: %q", rec.Body.String())
	}
}

func TestTopics(t *testing.T) {
	rec := do(newTestHandler(t), "GET", "/api/topics", "")
	ts := decode[[]store.TopicSummary](t, rec)
	if rec.Code != 200 || len(ts) != 1 || ts[0].Slug != "wehrpflicht" || ts[0].People != 1 || ts[0].Entries != 2 {
		t.Errorf("topics = %+v", ts)
	}
}

func TestPoliticianSearch(t *testing.T) {
	h := newTestHandler(t)
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"", []string{"Jürgen Müller", "Erika Ohnethema"}},
		{"?q=mueller", []string{"Jürgen Müller"}},
		{"?q=MÜLL", []string{"Jürgen Müller"}},
		{"?q=erika", []string{"Erika Ohnethema"}},
		{"?q=niemand", nil},
		{"?topic=wehrpflicht", []string{"Jürgen Müller"}},
	} {
		rec := do(h, "GET", "/api/politicians"+tc.query, "")
		ps := decode[[]store.PoliticianSummary](t, rec)
		var got []string
		for _, p := range ps {
			got = append(got, p.Name)
		}
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q: got %v, want %v", tc.query, got, tc.want)
		}
	}
	if rec := do(h, "GET", "/api/politicians?topic=mieten", ""); rec.Code != 404 {
		t.Errorf("unknown topic: %d", rec.Code)
	}
}

func TestPoliticianDetail(t *testing.T) {
	h := newTestHandler(t)
	rec := do(h, "GET", "/api/politicians/11000001", "")
	d := decode[store.PoliticianDetail](t, rec)
	if rec.Code != 200 || d.Name != "Jürgen Müller" || !d.IsMdB || len(d.Factions) != 1 || len(d.Topics) != 1 || d.Topics[0].Entries != 2 {
		t.Errorf("detail = %+v", d)
	}
	rec = do(h, "GET", "/api/politicians/99000001", "")
	d = decode[store.PoliticianDetail](t, rec)
	if d.IsMdB || len(d.Factions) != 0 || len(d.Topics) != 0 {
		t.Errorf("non-MdB detail = %+v", d)
	}
	if rec := do(h, "GET", "/api/politicians/nope", ""); rec.Code != 404 || !strings.Contains(rec.Body.String(), "unknown_politician") {
		t.Errorf("missing: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTimeline(t *testing.T) {
	h := newTestHandler(t)
	rec := do(h, "GET", "/api/timeline?politician=11000001&topic=wehrpflicht", "")
	tl := decode[TimelineResponse](t, rec)
	if rec.Code != 200 || len(tl.Entries) != 2 || tl.Changes != 1 {
		t.Fatalf("timeline = %d %+v", rec.Code, tl)
	}
	first, second := tl.Entries[0], tl.Entries[1]
	if first.Stance != stance.For || second.Stance != stance.Against || !second.Change || second.PreviousStance != stance.For {
		t.Errorf("entries = %+v", tl.Entries)
	}
	if first.Page != 5500 || first.Faction != "CDU/CSU" || first.Quote == "" || !strings.HasSuffix(first.SourceURL, ".pdf") {
		t.Errorf("first entry = %+v", first)
	}
	if tl.Topic.Name != "Wehrpflicht" || tl.Politician.Name != "Jürgen Müller" {
		t.Errorf("header = %+v %+v", tl.Topic, tl.Politician)
	}

	rec = do(h, "GET", "/api/timeline?politician=99000001&topic=wehrpflicht", "")
	if tl := decode[TimelineResponse](t, rec); len(tl.Entries) != 0 || tl.Entries == nil {
		t.Errorf("empty timeline must be [], got %s", rec.Body.String())
	}
	for _, q := range []string{"?politician=11000001&topic=x", "?politician=x&topic=wehrpflicht", ""} {
		if rec := do(h, "GET", "/api/timeline"+q, ""); rec.Code != 404 {
			t.Errorf("%q: %d", q, rec.Code)
		}
	}
}

func TestUnknownAPIRoute(t *testing.T) {
	rec := do(newTestHandler(t), "GET", "/api/nope", "")
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), "not_found") {
		t.Errorf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestReports(t *testing.T) {
	h := newTestHandler(t)
	valid := `{"paragraph_id":1,"topic":"wehrpflicht","message":"Das Zitat ist aus dem Zusammenhang gerissen.","contact":""}`
	for _, tc := range []struct {
		name, body, code string
		status           int
	}{
		{"valid", valid, "", 201},
		{"broken json", `{"paragraph_id":`, "invalid_json", 400},
		{"unknown field", `{"paragraph_id":1,"topic":"wehrpflicht","message":"lang genug hier","x":1}`, "invalid_json", 400},
		{"short", `{"paragraph_id":1,"topic":"wehrpflicht","message":"kurz"}`, "message_too_short", 422},
		{"long", `{"paragraph_id":1,"topic":"wehrpflicht","message":"` + strings.Repeat("ä", MaxMessage+1) + `"}`, "message_too_long", 422},
	} {
		// Each case comes from its own address so the rate limit stays out
		// of the way.
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/reports", strings.NewReader(tc.body))
		req.RemoteAddr = "198.51.100." + tc.name + ":1"
		h.ServeHTTP(rec, req)
		if rec.Code != tc.status || (tc.code != "" && !strings.Contains(rec.Body.String(), tc.code)) {
			t.Errorf("%s: %d %s", tc.name, rec.Code, rec.Body.String())
		}
	}
	for name, body := range map[string]string{
		"contact":        `{"paragraph_id":1,"topic":"wehrpflicht","message":"lang genug hier","contact":"` + strings.Repeat("x", MaxContact+1) + `"}`,
		"unknown topic":  `{"paragraph_id":1,"topic":"mieten","message":"lang genug hier"}`,
		"unknown para":   `{"paragraph_id":999,"topic":"wehrpflicht","message":"lang genug hier"}`,
		"zero paragraph": `{"paragraph_id":0,"topic":"wehrpflicht","message":"lang genug hier"}`,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/reports", strings.NewReader(body))
		req.RemoteAddr = "203.0.113." + name + ":1"
		h.ServeHTTP(rec, req)
		if rec.Code != 422 {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
}

func TestReportRateLimit(t *testing.T) {
	h := newTestHandler(t)
	body := `{"paragraph_id":1,"topic":"wehrpflicht","message":"Das Zitat ist aus dem Zusammenhang gerissen."}`
	for i := range ReportsPerIP {
		if rec := do(h, "POST", "/api/reports", body); rec.Code != 201 {
			t.Fatalf("report %d: %d", i, rec.Code)
		}
	}
	rec := do(h, "POST", "/api/reports", body)
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Errorf("over limit: %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestLimiterWindow(t *testing.T) {
	now := time.Unix(0, 0)
	l := newLimiter(1, time.Minute, func() time.Time { return now })
	if !l.allow("a") || l.allow("a") || !l.allow("b") {
		t.Fatal("first window wrong")
	}
	now = now.Add(time.Minute)
	if !l.allow("a") {
		t.Error("new window must allow again")
	}
	if len(l.seen) != 1 {
		t.Errorf("expired keys kept: %d", len(l.seen))
	}
}

func TestFold(t *testing.T) {
	if fold("Größe Ähre Öl Üben") != "groesse aehre oel ueben" {
		t.Errorf("fold = %q", fold("Größe Ähre Öl Üben"))
	}
}

func TestAPIOnly(t *testing.T) {
	h := NewHandler(&Server{Store: seed(t)})
	if rec := do(h, "GET", "/", ""); rec.Code != 404 {
		t.Errorf("without frontend / = %d", rec.Code)
	}
}

func TestEvaluation(t *testing.T) {
	st := seed(t)
	h := NewHandler(&Server{Store: st})
	if rec := do(h, "GET", "/api/evaluation?topic=wehrpflicht", ""); rec.Code != 404 || !strings.Contains(rec.Body.String(), "no_evaluation") {
		t.Errorf("without a reviewed run: %d %s", rec.Code, rec.Body.String())
	}
	if err := st.SaveEvaluation(ctx, store.Evaluation{
		Topic: "wehrpflicht", Classifier: "fake", PromptVersion: stance.PromptVersion,
		Items: 45, StanceAccuracy: 0.75, GoldReviewed: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	rec := do(h, "GET", "/api/evaluation?topic=wehrpflicht", "")
	if e := decode[store.Evaluation](t, rec); rec.Code != 200 || e.StanceAccuracy != 0.75 {
		t.Errorf("published: %d %+v", rec.Code, e)
	}
}
