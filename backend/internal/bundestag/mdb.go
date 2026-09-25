package bundestag

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Politician is one member of the Bundestag from the MdB master data.
type Politician struct {
	ID     string
	Title  string
	First  string
	Last   string
	Affix  string
	Party  string
	Gender string
	// Memberships lists faction memberships per legislative period.
	Memberships []Membership
}

// Name is the display name.
func (p Politician) Name() string {
	return Speaker{Title: p.Title, First: p.First, Last: p.Last, Suffix: p.Affix}.Name()
}

// Membership is a faction membership within one legislative period. To is
// zero while it lasts.
type Membership struct {
	Period  int
	Faction string
	From    time.Time
	To      time.Time
}

type mdbXML struct {
	ID    string `xml:"ID"`
	Namen []struct {
		Nachname string `xml:"NACHNAME"`
		Vorname  string `xml:"VORNAME"`
		Adel     string `xml:"ADEL"`
		Praefix  string `xml:"PRAEFIX"`
		Titel    string `xml:"ANREDE_TITEL"`
		Bis      string `xml:"HISTORIE_BIS"`
	} `xml:"NAMEN>NAME"`
	Partei     string `xml:"BIOGRAFISCHE_ANGABEN>PARTEI_KURZ"`
	Geschlecht string `xml:"BIOGRAFISCHE_ANGABEN>GESCHLECHT"`
	Perioden   []struct {
		WP           int    `xml:"WP"`
		Von          string `xml:"MDBWP_VON"`
		Bis          string `xml:"MDBWP_BIS"`
		Institutions []struct {
			Art string `xml:"INSART_LANG"`
			Ins string `xml:"INS_LANG"`
			Von string `xml:"MDBINS_VON"`
			Bis string `xml:"MDBINS_BIS"`
		} `xml:"INSTITUTIONEN>INSTITUTION"`
	} `xml:"WAHLPERIODEN>WAHLPERIODE"`
}

// ReadMdBZip reads MDB_STAMMDATEN.XML from the archive the Bundestag
// publishes.
func ReadMdBZip(path string, minPeriod int) ([]Politician, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "MDB_STAMMDATEN.XML" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return ParseMdB(rc, minPeriod)
	}
	return nil, fmt.Errorf("%s: no MDB_STAMMDATEN.XML in archive", path)
}

// ParseMdB reads the master data and keeps members who sat in minPeriod or
// later, with memberships from minPeriod on.
func ParseMdB(r io.Reader, minPeriod int) ([]Politician, error) {
	dec := xml.NewDecoder(r)
	var out []Politician
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "MDB" {
			continue
		}
		var m mdbXML
		if err := dec.DecodeElement(&m, &se); err != nil {
			return nil, err
		}
		p, keep, err := toPolitician(m, minPeriod)
		if err != nil {
			return nil, fmt.Errorf("MdB %s: %w", m.ID, err)
		}
		if keep {
			out = append(out, p)
		}
	}
}

func toPolitician(m mdbXML, minPeriod int) (Politician, bool, error) {
	p := Politician{ID: m.ID, Party: trim(m.Partei), Gender: trim(m.Geschlecht)}
	// The current name is the one without an end date; earlier names stay in
	// the history.
	for _, n := range m.Namen {
		if trim(n.Bis) == "" || p.Last == "" {
			p.First, p.Last, p.Title = trim(n.Vorname), trim(n.Nachname), trim(n.Titel)
			p.Affix = strings.TrimSpace(trim(n.Adel) + " " + trim(n.Praefix))
		}
	}
	keep := false
	for _, wp := range m.Perioden {
		if wp.WP < minPeriod {
			continue
		}
		keep = true
		for _, ins := range wp.Institutions {
			if ins.Art != "Fraktion/Gruppe" {
				continue
			}
			from, err := date(firstNonEmpty(ins.Von, wp.Von))
			if err != nil {
				return p, false, err
			}
			to, err := date(firstNonEmpty(ins.Bis, wp.Bis))
			if err != nil {
				return p, false, err
			}
			p.Memberships = append(p.Memberships, Membership{Period: wp.WP, Faction: NormalizeFaction(ins.Ins), From: from, To: to})
		}
	}
	return p, keep, nil
}

func firstNonEmpty(a, b string) string {
	if trim(a) != "" {
		return a
	}
	return b
}

func date(s string) (time.Time, error) {
	if s = trim(s); s == "" {
		return time.Time{}, nil
	}
	return time.Parse("02.01.2006", s)
}

// Canonical faction names, as the website shows them.
const (
	CDUCSU      = "CDU/CSU"
	SPD         = "SPD"
	AfD         = "AfD"
	Greens      = "BÜNDNIS 90/DIE GRÜNEN"
	FDP         = "FDP"
	Left        = "Die Linke"
	BSW         = "BSW"
	NonAttached = "fraktionslos"
)

// NormalizeFaction maps the many spellings in protocols and master data to
// one canonical name, or "" when the text is not a faction (the protocols
// occasionally put things like "zur Geschäftsordnung" in that element).
func NormalizeFaction(s string) string {
	key := strings.ToUpper(strings.Join(strings.Fields(s), ""))
	switch {
	case key == "CDU/CSU" || strings.Contains(key, "CHRISTLICHDEMOKRATISCHEN"):
		return CDUCSU
	case key == "SPD" || key == "SPDSPD" || strings.Contains(key, "SOZIALDEMOKRATISCHEN"):
		return SPD
	case key == "AFD" || strings.Contains(key, "ALTERNATIVEFÜRDEUTSCHLAND"):
		return AfD
	case strings.HasPrefix(key, "BÜNDNIS90/") || strings.Contains(key, "BÜNDNIS90/DIEGRÜNEN"):
		return Greens
	case key == "FDP" || strings.Contains(key, "FREIENDEMOKRATISCHEN"):
		return FDP
	case strings.Contains(key, "DIELINKE"):
		return Left
	case key == "BSW" || strings.Contains(key, "WAGENKNECHT"):
		return BSW
	case key == "FRAKTIONSLOS":
		return NonAttached
	}
	return ""
}

// FactionOn returns the politician's faction on day, or "".
func (p Politician) FactionOn(day time.Time) string {
	for _, m := range p.Memberships {
		if !day.Before(m.From) && (m.To.IsZero() || !day.After(m.To)) {
			return m.Faction
		}
	}
	return ""
}

// ParsePeriods reads a comma-separated list such as "19,20,21".
func ParsePeriods(s string) ([]int, error) {
	var out []int
	for _, f := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n < 19 {
			return nil, fmt.Errorf("legislative period %q: only 19 and later have structured protocols", f)
		}
		out = append(out, n)
	}
	return out, nil
}
