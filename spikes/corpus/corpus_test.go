package corpus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	f, err := os.Open("testdata/sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2021, 10, 26, 0, 0, 0, 0, time.UTC)
	want := []Speech{
		{
			ID: "ID2000700100", Period: 20, Session: 7, Date: date,
			SpeakerID: "11004325", Speaker: "Gabriele Katzmarek", Party: "SPD",
			Paragraphs: []string{"Sehr geehrter Herr Präsident!", "Die Wehrpflicht bleibt ausgesetzt."},
		},
		{
			ID: "ID2000700200", Period: 20, Session: 7, Date: date,
			SpeakerID: "11005000", Speaker: "Dr. Maria Muster", Party: "Bundesministerin der Verteidigung",
			Paragraphs: []string{"Wir brauchen einen neuen Wehrdienst."},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseRejectsBadDate(t *testing.T) {
	_, err := Parse(strings.NewReader(`<dbtplenarprotokoll sitzung-datum="2021-10-26"/>`))
	if err == nil {
		t.Fatal("want error for ISO date")
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "20")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "20007.xml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d speeches, want 2", len(got))
	}
}

func TestProtocolURL(t *testing.T) {
	if got, want := ProtocolURL(20, 1), "https://dserver.bundestag.de/btp/20/20001.xml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
