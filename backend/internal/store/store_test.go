package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var (
	ctx = context.Background()
	now = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func protocol() bundestag.Protocol {
	return bundestag.Protocol{
		Period: 20, Session: 7, Date: day(2024, 1, 10),
		Speeches: []bundestag.Speech{
			{
				ID: "ID2000700100", Page: 101,
				Speaker: bundestag.Speaker{ID: "11004000", First: "Berta", Last: "Wechsel", Faction: "zur Geschäftsordnung"},
				Paragraphs: []bundestag.Paragraph{
					{Text: "Die Wehrpflicht bleibt ausgesetzt.", Page: 101},
					{Text: "Ein Zitat.", Quote: true, Page: 102},
				},
			},
			{
				ID:         "ID2000700200",
				Speaker:    bundestag.Speaker{ID: "99000001", First: "Boris", Last: "Minister", Role: "Bundesminister der Verteidigung"},
				Paragraphs: []bundestag.Paragraph{{Text: "Wir brauchen einen neuen Wehrdienst."}},
			},
		},
	}
}

func politicians() []bundestag.Politician {
	return []bundestag.Politician{{
		ID: "11004000", First: "Berta", Last: "Wechsel", Party: "BSW",
		Memberships: []bundestag.Membership{
			{Period: 20, Faction: bundestag.Left, From: day(2021, 10, 26), To: day(2023, 12, 5)},
			{Period: 20, Faction: bundestag.BSW, From: day(2023, 12, 6), To: day(2025, 3, 25)},
		},
	}}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	for range 2 {
		s, err := Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		v, err := s.SchemaVersion(ctx)
		if err != nil || v != 2 {
			t.Fatalf("version = %d, %v", v, err)
		}
		s.Close()
	}
}

func TestOpenInMemory(t *testing.T) {
	s, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Stats(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSaveProtocol(t *testing.T) {
	s := openTest(t)
	if err := s.UpsertPoliticians(ctx, politicians()); err != nil {
		t.Fatal(err)
	}
	p := protocol()
	if err := s.SaveProtocol(ctx, p, "https://example.test/20007.xml", now); err != nil {
		t.Fatal(err)
	}

	st, err := s.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st != (Stats{Politicians: 2, Sessions: 1, Speeches: 2, Paragraphs: 3}) {
		t.Errorf("stats = %+v", st)
	}
	if ok, _ := s.HasSession(ctx, 20, 7); !ok {
		t.Error("HasSession(20, 7) = false")
	}
	if ok, _ := s.HasSession(ctx, 20, 8); ok {
		t.Error("HasSession(20, 8) = true")
	}

	var faction, role string
	var page any
	if err := s.db.QueryRow(`SELECT faction, role, page FROM speeches WHERE id = 'ID2000700100'`).Scan(&faction, &role, &page); err != nil {
		t.Fatal(err)
	}
	if faction != bundestag.BSW {
		t.Errorf("faction = %q, want %q from the master data on the session day", faction, bundestag.BSW)
	}
	if err := s.db.QueryRow(`SELECT faction, role, page FROM speeches WHERE id = 'ID2000700200'`).Scan(&faction, &role, &page); err != nil {
		t.Fatal(err)
	}
	if faction != "" || role != "Bundesminister der Verteidigung" || page != nil {
		t.Errorf("minister speech = %q %q %v", faction, role, page)
	}
	var isMdB int
	if err := s.db.QueryRow(`SELECT is_mdb FROM politicians WHERE id = '99000001'`).Scan(&isMdB); err != nil || isMdB != 0 {
		t.Errorf("minister is_mdb = %d, %v", isMdB, err)
	}
}

func TestSaveProtocolAgainKeepsParagraphIDs(t *testing.T) {
	s := openTest(t)
	p := protocol()
	if err := s.SaveProtocol(ctx, p, "u", now); err != nil {
		t.Fatal(err)
	}
	id := func(pos int) int64 {
		var id int64
		if err := s.db.QueryRow(`SELECT id FROM paragraphs WHERE speech_id = 'ID2000700100' AND position = ?`, pos).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	before := id(0)

	p.Speeches[0].Paragraphs = []bundestag.Paragraph{{Text: "Die Wehrpflicht bleibt ausgesetzt, korrigiert."}}
	if err := s.SaveProtocol(ctx, p, "u", now); err != nil {
		t.Fatal(err)
	}
	if after := id(0); after != before {
		t.Errorf("paragraph id changed from %d to %d", before, after)
	}
	st, _ := s.Stats(ctx)
	if st.Paragraphs != 2 || st.Sessions != 1 {
		t.Errorf("stats after re-import = %+v, want the dropped paragraph removed", st)
	}
	if n := search(t, s, "korrigiert"); n != 1 {
		t.Errorf("full-text index not updated: %d hits", n)
	}
	if n := search(t, s, "Zitat"); n != 0 {
		t.Errorf("deleted paragraph still indexed: %d hits", n)
	}
}

func search(t *testing.T, s *Store, q string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM paragraphs_fts WHERE paragraphs_fts MATCH ?`, q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFullTextSearchPrefixAndUmlauts(t *testing.T) {
	s := openTest(t)
	if err := s.SaveProtocol(ctx, protocol(), "u", now); err != nil {
		t.Fatal(err)
	}
	if n := search(t, s, "wehrpflicht*"); n != 1 {
		t.Errorf("prefix search = %d", n)
	}
	if n := search(t, s, "wehrdienst"); n != 1 {
		t.Errorf("case-insensitive search = %d", n)
	}
}

func TestUpsertPoliticiansReplacesMemberships(t *testing.T) {
	s := openTest(t)
	ps := politicians()
	if err := s.UpsertPoliticians(ctx, ps); err != nil {
		t.Fatal(err)
	}
	ps[0].Memberships = ps[0].Memberships[:1]
	ps[0].Last = "Neu"
	if err := s.UpsertPoliticians(ctx, ps); err != nil {
		t.Fatal(err)
	}
	var n int
	var last string
	s.db.QueryRow(`SELECT count(*) FROM memberships`).Scan(&n)
	s.db.QueryRow(`SELECT last_name FROM politicians`).Scan(&last)
	if n != 1 || last != "Neu" {
		t.Errorf("memberships = %d, last name = %q", n, last)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	s := openTest(t)
	_, err := s.db.Exec(`INSERT INTO paragraphs (speech_id, position, text) VALUES ('missing', 0, 'x')`)
	if err == nil {
		t.Fatal("want foreign key violation")
	}
}

func TestSchemaVersionIsLatest(t *testing.T) {
	s := openTest(t)
	v, err := s.SchemaVersion(ctx)
	if err != nil || v != 2 {
		t.Fatalf("version = %d, %v", v, err)
	}
}

func TestStanceRejectsUnknownLabel(t *testing.T) {
	s := openTest(t)
	if err := s.SaveProtocol(ctx, protocol(), "u", now); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO topics VALUES ('t', 'T', 'Q?', 'x')`); err != nil {
		t.Fatal(err)
	}
	_, err := s.db.Exec(`INSERT INTO stances VALUES (1, 't', 1, 'ja', '', 0, '', 0, 'm', 'v1', '')`)
	if err == nil {
		t.Fatal("CHECK constraint on stance not enforced")
	}
}

func TestTopicAssignmentAndPending(t *testing.T) {
	s := openTest(t)
	if err := s.SaveProtocol(ctx, protocol(), "u", now); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncTopics(ctx, topic.All); err != nil {
		t.Fatal(err)
	}
	n, err := s.AssignTopic(ctx, topic.Wehrpflicht)
	if err != nil || n != 2 {
		t.Fatalf("AssignTopic = %d, %v", n, err)
	}
	pending, err := s.Pending(ctx, "wehrpflicht", 0)
	if err != nil || len(pending) != 2 {
		t.Fatalf("Pending = %v, %v", pending, err)
	}
	if limited, _ := s.Pending(ctx, "wehrpflicht", 1); len(limited) != 1 {
		t.Errorf("limit ignored: %d", len(limited))
	}

	a := stance.Answer{Relevant: true, Stance: stance.Against, Quote: "q", Rationale: "r", Confidence: 0.7}
	if err := s.SaveStance(ctx, pending[0].ParagraphID, "wehrpflicht", a, false, "fake", now); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.Pending(ctx, "wehrpflicht", 0); len(left) != 1 {
		t.Errorf("classified paragraph still pending: %d left", len(left))
	}
	// A result from an older prompt counts as pending again.
	s.db.Exec(`UPDATE stances SET prompt_version = 'v0'`)
	if left, _ := s.Pending(ctx, "wehrpflicht", 0); len(left) != 2 {
		t.Errorf("outdated prompt not pending: %d", len(left))
	}
	if err := s.DeleteStances(ctx, "wehrpflicht"); err != nil {
		t.Fatal(err)
	}
	var cnt int
	s.db.QueryRow(`SELECT count(*) FROM stances`).Scan(&cnt)
	if cnt != 0 {
		t.Errorf("stances left: %d", cnt)
	}
}
