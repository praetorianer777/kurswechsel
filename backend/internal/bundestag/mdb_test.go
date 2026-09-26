package bundestag

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestParseMdB(t *testing.T) {
	f, err := os.Open("testdata/mdb.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ParseMdB(f, 19)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d politicians, want 1 (the WP17-only member is dropped)", len(got))
	}
	p := got[0]
	if p.ID != "11004000" || p.Name() != "Dr. Berta Wechsel Freifrau von" || p.Party != "DIE LINKE." || p.Gender != "weiblich" {
		t.Errorf("politician = %+v, name %q", p, p.Name())
	}
	want := []Membership{
		{Period: 20, Faction: Left, From: day(2021, 10, 26), To: day(2023, 12, 5)},
		// An open membership ends with the legislative period.
		{Period: 20, Faction: BSW, From: day(2023, 12, 6), To: day(2025, 3, 25)},
	}
	if len(p.Memberships) != 2 || p.Memberships[0] != want[0] || p.Memberships[1] != want[1] {
		t.Errorf("memberships = %+v", p.Memberships)
	}
	for d, want := range map[time.Time]string{
		day(2022, 1, 1):   Left,
		day(2023, 12, 5):  Left,
		day(2023, 12, 6):  BSW,
		day(2025, 3, 25):  BSW,
		day(2025, 3, 26):  "",
		day(2020, 12, 31): "",
	} {
		if got := p.FactionOn(d); got != want {
			t.Errorf("FactionOn(%s) = %q, want %q", d.Format(time.DateOnly), got, want)
		}
	}
}

func TestReadMdBZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MdB-Stammdaten.zip")
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	w, err := zw.Create("MDB_STAMMDATEN.XML")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/mdb.xml")
	if err != nil {
		t.Fatal(err)
	}
	w.Write(data)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	out.Close()

	got, err := ReadMdBZip(path, 19)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %d politicians", len(got))
	}
	if _, err := ReadMdBZip(filepath.Join(dir, "missing.zip"), 19); err == nil {
		t.Error("want error for missing file")
	}
}

func TestParseMdBBadDate(t *testing.T) {
	const in = `<DOCUMENT><MDB><ID>1</ID><WAHLPERIODEN><WAHLPERIODE><WP>20</WP><MDBWP_VON>2021-10-26</MDBWP_VON>
<INSTITUTIONEN><INSTITUTION><INSART_LANG>Fraktion/Gruppe</INSART_LANG><INS_LANG>Fraktion der FDP</INS_LANG></INSTITUTION></INSTITUTIONEN>
</WAHLPERIODE></WAHLPERIODEN></MDB></DOCUMENT>`
	if _, err := ParseMdB(strings.NewReader(in), 19); err == nil {
		t.Fatal("want error")
	}
}

func TestNormalizeFaction(t *testing.T) {
	for in, want := range map[string]string{
		"CDU/CSU":  CDUCSU,
		"CDU/ CSU": CDUCSU,
		"Fraktion der Christlich Demokratischen Union/Christlich - Sozialen Union": CDUCSU,
		"SPD  ":                   SPD,
		"SPDSPD":                  SPD,
		" AfD":                    AfD,
		"BÜNDNIS 90/DIE GRÜNEN":   Greens,
		"BÜNDNIS 90/  DIE GRÜNEN": Greens,
		"BÜNDNIS 90/DIE GRÜNE N":  Greens,
		"BÜNDNIS 90/":             Greens,
		"Bündnis 90/Die Grünen":   Greens,
		"Fraktion der Freien Demokratischen Partei": FDP,
		"DIE LINKE":           Left,
		"Die Linke":           Left,
		"DIE LINKE T":         Left,
		"Fraktion DIE LINKE.": Left,
		"Gruppe Die Linke":    Left,
		"Gruppe BSW - Bündnis Sahra Wagenknecht - Vernunft und Gerechtigkeit": BSW,
		"Fraktionslos":         NonAttached,
		"fraktionslos":         NonAttached,
		"zur Geschäftsordnung": "",
		"SPDCDU/CSU":           "",
		"Bremen":               "",
	} {
		if got := NormalizeFaction(in); got != want {
			t.Errorf("NormalizeFaction(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePeriods(t *testing.T) {
	got, err := ParsePeriods("19, 20,21")
	if err != nil || len(got) != 3 || got[2] != 21 {
		t.Errorf("got %v, %v", got, err)
	}
	for _, bad := range []string{"18", "x", "20,"} {
		if _, err := ParsePeriods(bad); err == nil {
			t.Errorf("ParsePeriods(%q): want error", bad)
		}
	}
}
