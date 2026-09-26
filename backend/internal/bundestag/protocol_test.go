package bundestag

import (
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files")

func parseFile(t *testing.T, path string) Protocol {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := ParseProtocol(f)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A real speech from session 20/3: the presiding officer's closing line and
// all interjections must not end up in the speaker's paragraphs.
func TestParseProtocolGolden(t *testing.T) {
	got := parseFile(t, "testdata/rede-excerpt.xml")
	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/rede-excerpt.golden.json"
	if *update {
		if err := os.WriteFile(golden, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(b)+"\n" {
		t.Errorf("parse result differs from %s; run go test -update and review the diff", golden)
	}
	for _, sp := range got.Speeches {
		for _, p := range sp.Paragraphs {
			if strings.Contains(p.Text, "Als nächster Redner") || strings.Contains(p.Text, "Beifall") {
				t.Errorf("foreign text in %s: %q", sp.ID, p.Text)
			}
		}
	}
}

func TestParseProtocolEdgeCases(t *testing.T) {
	got := parseFile(t, "testdata/edge-cases.xml")

	if got.Period != 21 || got.Session != 42 || !got.Date.Equal(time.Date(2025, 12, 5, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("header = %d/%d %v", got.Period, got.Session, got.Date)
	}
	want := []Speech{
		{
			ID:   "ID214204200",
			Page: 4101,
			Speaker: Speaker{
				ID: "11005001", Title: "Dr.", First: "Erika", Last: "Beispiel", Suffix: "von", Faction: "SPD",
			},
			Paragraphs: []Paragraph{
				{Text: "Der CO2-Ausstoß sinkt.", Page: 4101},
				{Text: "Zeile mit Umbruch und geschütztem Leerzeichen.", Page: 4101},
				{Text: "Ein Zitat eines Gegners.", Quote: true, Page: 4102},
				{Text: "Äußerer Absatz mit innerem Ende.", Page: 4102},
				{Text: "Meine Antwort.", Page: 4102},
			},
		},
		{
			ID:         "ID214204300",
			Page:       4102,
			Speaker:    Speaker{ID: "99000001", First: "Boris", Last: "Minister", Role: "Bundesminister der Verteidigung"},
			Paragraphs: []Paragraph{{Text: "Kurz und knapp.", Page: 4102}},
		},
	}
	if !reflect.DeepEqual(got.Speeches, want) {
		gb, _ := json.MarshalIndent(got.Speeches, "", "  ")
		t.Errorf("speeches:\n%s", gb)
	}
}

func TestSpeakerName(t *testing.T) {
	s := Speaker{Title: "Dr.", First: "Erika", Last: "Beispiel", Suffix: "von"}
	if got := s.Name(); got != "Dr. Erika Beispiel von" {
		t.Errorf("Name() = %q", got)
	}
	if got := (Speaker{First: "Max", Last: "Muster"}).Name(); got != "Max Muster" {
		t.Errorf("Name() = %q", got)
	}
}

func TestParseProtocolErrors(t *testing.T) {
	for name, in := range map[string]string{
		"not a protocol":  `<foo/>`,
		"broken xml":      `<dbtplenarprotokoll wahlperiode="20"`,
		"bad date":        `<dbtplenarprotokoll wahlperiode="20" sitzung-nr="1" sitzung-datum="2021-10-26"/>`,
		"bad period":      `<dbtplenarprotokoll wahlperiode="XX" sitzung-nr="1" sitzung-datum="26.10.2021"/>`,
		"missing session": `<dbtplenarprotokoll wahlperiode="20" sitzung-datum="26.10.2021"/>`,
		"bad session":     `<dbtplenarprotokoll wahlperiode="20" sitzung-nr="x" sitzung-datum="26.10.2021"/>`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseProtocol(strings.NewReader(in)); err == nil {
				t.Error("want error")
			}
		})
	}
}
